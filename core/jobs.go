package core

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
	"sort"
	"strings"
	"syscall"
	"time"
)

// Job is a background agent run. Its stdout (Claude's stream-json) goes to <id>.jsonl, and the
// process runs in its own session so it outlives gh-prs.
type Job struct {
	ID      string    `json:"id"`                // also the Claude session id of a new session
	Session string    `json:"session,omitempty"` // the Claude session a resumed run continues
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

const maxActivity = 100

// jobView is a Job plus what its log says so far. Logs are read incrementally.
type JobView struct {
	Job
	Status   JobStatus
	Activity []string // recent tool calls and messages, newest last
	Result   string
	Cost     float64
	Turns    int
	Finished time.Time
	Waiting  bool     // a /loop job between rounds: it reported a result but is still running
	Stopped  bool     // you stopped it
	Rounds   []string // the headline of each result, oldest first: one per /loop round

	over    bool // the process exited and its log is fully read
	offset  int64
	partial []byte
}

// SessionID is the Claude session the job ran in, to resume it.
func (j Job) SessionID() string {
	if j.Session != "" {
		return j.Session
	}
	return j.ID
}

func PrKey(repo string, number int) string { return fmt.Sprintf("%s#%d", repo, number) }

func JobsDir() string { return filepath.Join(cacheDir(), "jobs") }

func (j Job) Path(ext string) string { return filepath.Join(JobsDir(), j.ID+ext) }

// newJobID returns a random UUID v4, the format Claude requires for --session-id.
func NewJobID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func StartJob(pp Prepared) (*JobView, error) {
	j := Job{ID: pp.jobID, Session: pp.session, Key: pp.Cmd.Key, Name: pp.Cmd.Name, Repo: pp.Pr.Repo, Number: pp.Pr.Number, Dir: pp.dir, Started: time.Now()}
	if err := os.MkdirAll(JobsDir(), 0o700); err != nil {
		return nil, err
	}
	out, err := os.OpenFile(j.Path(".jsonl"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, err
	}
	defer out.Close()
	errf, err := os.OpenFile(j.Path(".err"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, err
	}
	defer errf.Close()

	c := pp.ExecCmd()
	c.Stdout, c.Stderr = out, errf
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true} // survives gh-prs quitting and ctrl+c
	if err := c.Start(); err != nil {
		return nil, err
	}
	go c.Wait() //nolint:errcheck // reap it; status comes from the log
	j.PID = c.Process.Pid

	meta, _ := json.Marshal(j)
	if err := os.WriteFile(j.Path(".json"), meta, 0o600); err != nil {
		return nil, err
	}
	return &JobView{Job: j, Status: JobRunning}, nil
}

// loadJobs returns the latest job per PR.
func LoadJobs() map[string]*JobView {
	jobs := map[string]*JobView{}
	for _, j := range readJobs() {
		k := PrKey(j.Repo, j.Number)
		if cur := jobs[k]; cur == nil || j.Started.After(cur.Started) {
			jobs[k] = &JobView{Job: j}
		}
	}
	for _, v := range jobs {
		v.Update()
	}
	return jobs
}

// LoadAllJobs returns the newest jobs across all PRs, newest first, at most limit.
func LoadAllJobs(limit int) []*JobView {
	all := readJobs()
	sort.Slice(all, func(a, b int) bool { return all[a].Started.After(all[b].Started) })
	var out []*JobView
	for _, j := range all[:min(limit, len(all))] {
		v := &JobView{Job: j}
		v.Update()
		out = append(out, v)
	}
	return out
}

func readJobs() []Job {
	var jobs []Job
	files, _ := filepath.Glob(filepath.Join(JobsDir(), "*.json"))
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var j Job
		if json.Unmarshal(raw, &j) == nil && j.ID != "" {
			jobs = append(jobs, j)
		}
	}
	return jobs
}

// alive reports whether the job's process still runs. The age limit guards against a reused PID after a reboot.
func (v *JobView) alive() bool {
	if time.Since(v.Started) > 12*time.Hour {
		return false
	}
	err := syscall.Kill(v.PID, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

// update reads new log lines. Liveness is checked before reading so a run that ends between
// the two isn't mistaken for one that died without a result.
func (v *JobView) Update() {
	if v.over {
		return
	}
	dead := !v.alive()
	if f, err := os.Open(v.Path(".jsonl")); err == nil {
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
		if b, err := os.ReadFile(v.Path(".err")); err == nil {
			if l := LastLine(string(b)); l != "" {
				v.Result = l
			}
		}
	}
	if _, err := os.Stat(v.Path(".stopped")); err == nil && dead {
		v.Status, v.Stopped, v.Result = JobFailed, true, "stopped by you"
	}
	v.Waiting, v.over = !dead && v.Status != JobRunning, dead
	if v.Status != JobRunning && !v.Waiting && v.Finished.IsZero() {
		v.Finished = time.Now()
		if info, err := os.Stat(v.Path(".jsonl")); err == nil && dead {
			v.Finished = info.ModTime()
		}
	}
}

// Stop ends the job and everything it started: it leads its own process group. A marker file
// records that you stopped it, so it isn't shown as a crash, also after a restart.
func (v *JobView) Stop() error {
	if !v.alive() {
		return fmt.Errorf("%s on #%d isn't running", v.Name, v.Number)
	}
	if err := os.WriteFile(v.Path(".stopped"), nil, 0o600); err != nil {
		return err
	}
	return syscall.Kill(-v.PID, syscall.SIGTERM)
}

// StopAndWait stops the job and waits up to d for it to exit, so its session can be resumed.
func (v *JobView) StopAndWait(d time.Duration) error {
	if err := v.Stop(); err != nil {
		return err
	}
	for end := time.Now().Add(d); time.Now().Before(end); time.Sleep(100 * time.Millisecond) {
		if !v.alive() {
			return nil
		}
	}
	return fmt.Errorf("%s on #%d didn't stop in time", v.Name, v.Number)
}

// StopAgent stops the agent running on a PR and says what it stopped.
func StopAgent(jobs map[string]*JobView, p PR) (string, error) {
	v := jobs[PrKey(p.Repo, p.Number)]
	if v == nil || !v.Active() {
		return "", fmt.Errorf("no agent is running on #%d", p.Number)
	}
	if err := v.Stop(); err != nil {
		return "", err
	}
	return fmt.Sprintf("stopped %s on #%d", v.Name, p.Number), nil
}

// active reports whether the job still runs, mid-round or waiting for its next one.
func (v *JobView) Active() bool { return v.Status == JobRunning || v.Waiting }

func (v *JobView) Elapsed() time.Duration {
	end := v.Finished
	if end.IsZero() {
		end = time.Now()
	}
	return end.Sub(v.Started)
}

func ShortDuration(d time.Duration) string {
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

func LastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(Sanitize(s)), "\n")
	return lines[len(lines)-1]
}
