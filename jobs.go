package main

// Background agent jobs: starting them detached, reloading them, reading their logs.

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// Job is a background agent run. Its stdout (Claude's stream-json) goes to <id>.jsonl, and the
// process runs in its own session so it outlives gh-prs.
type Job struct {
	ID      string    `json:"id"` // also the Claude session id, so the run can be resumed
	Key     string    `json:"key"`
	Name    string    `json:"name"`
	Repo    string    `json:"repo"`
	Number  int       `json:"number"`
	Dir     string    `json:"dir"`
	PID     int       `json:"pid"`
	Started time.Time `json:"started"`
}

type JobStatus int

const (
	JobRunning JobStatus = iota
	JobDone
	JobFailed
)

const maxActivity = 20

// jobView is a Job plus what its log says so far. Logs are read incrementally.
type jobView struct {
	Job
	Status   JobStatus
	Activity []string // recent tool calls and messages, newest last
	Result   string
	Cost     float64
	Turns    int
	Finished time.Time

	offset  int64
	partial []byte
}

func prKey(repo string, number int) string { return fmt.Sprintf("%s#%d", repo, number) }

func jobsDir() string { return filepath.Join(cacheDir(), "jobs") }

func (j Job) path(ext string) string { return filepath.Join(jobsDir(), j.ID+ext) }

// newJobID returns a random UUID v4, the format Claude requires for --session-id.
func newJobID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func startJob(pp prepared) (*jobView, error) {
	j := Job{ID: pp.jobID, Key: pp.cmd.Key, Name: pp.cmd.Name, Repo: pp.pr.Repo, Number: pp.pr.Number, Dir: pp.dir, Started: time.Now()}
	if err := os.MkdirAll(jobsDir(), 0o700); err != nil {
		return nil, err
	}
	out, err := os.OpenFile(j.path(".jsonl"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, err
	}
	defer out.Close()
	errf, err := os.OpenFile(j.path(".err"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, err
	}
	defer errf.Close()

	c := pp.execCmd()
	c.Stdout, c.Stderr = out, errf
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true} // survives gh-prs quitting and ctrl+c
	if err := c.Start(); err != nil {
		return nil, err
	}
	go c.Wait() //nolint:errcheck // reap it; status comes from the log
	j.PID = c.Process.Pid

	meta, _ := json.Marshal(j)
	if err := os.WriteFile(j.path(".json"), meta, 0o600); err != nil {
		return nil, err
	}
	return &jobView{Job: j, Status: JobRunning}, nil
}

// loadJobs returns the latest job per PR.
func loadJobs() map[string]*jobView {
	jobs := map[string]*jobView{}
	files, _ := filepath.Glob(filepath.Join(jobsDir(), "*.json"))
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var j Job
		if json.Unmarshal(raw, &j) != nil || j.ID == "" {
			continue
		}
		k := prKey(j.Repo, j.Number)
		if cur := jobs[k]; cur == nil || j.Started.After(cur.Started) {
			jobs[k] = &jobView{Job: j}
		}
	}
	for _, v := range jobs {
		v.update()
	}
	return jobs
}

// alive reports whether the job's process still runs. The age limit guards against a reused PID after a reboot.
func (v *jobView) alive() bool {
	if time.Since(v.Started) > 12*time.Hour {
		return false
	}
	err := syscall.Kill(v.PID, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

// update reads new log lines. Liveness is checked before reading so a run that ends between
// the two isn't mistaken for one that died without a result.
func (v *jobView) update() {
	if v.Status != JobRunning {
		return
	}
	dead := !v.alive()
	if f, err := os.Open(v.path(".jsonl")); err == nil {
		if _, err := f.Seek(v.offset, io.SeekStart); err == nil {
			data, _ := io.ReadAll(f)
			v.offset += int64(len(data))
			v.partial = append(v.partial, data...)
		}
		f.Close()
	}
	for {
		i := bytes.IndexByte(v.partial, '\n')
		if i < 0 {
			break
		}
		v.apply(v.partial[:i])
		v.partial = v.partial[i+1:]
	}
	if v.Status == JobRunning && dead {
		v.apply(v.partial) // a last line without a newline
		v.partial = nil
	}
	if v.Status == JobRunning && dead {
		v.Status = JobFailed
		v.Result = "exited without a result"
		if b, err := os.ReadFile(v.path(".err")); err == nil {
			if l := lastLine(string(b)); l != "" {
				v.Result = l
			}
		}
	}
	if v.Status != JobRunning && v.Finished.IsZero() {
		v.Finished = time.Now()
		if info, err := os.Stat(v.path(".jsonl")); err == nil && dead {
			v.Finished = info.ModTime()
		}
	}
}

func (v *jobView) elapsed() time.Duration {
	end := v.Finished
	if end.IsZero() {
		end = time.Now()
	}
	return end.Sub(v.Started)
}

func shortDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
