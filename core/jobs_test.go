package core

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"testing"
	"time"
)

func deadPID(t *testing.T) int {
	c := exec.Command("true")
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	return c.Process.Pid
}

func writeJob(t *testing.T, pid int, log, stderr string) *JobView {
	t.Helper()
	j := Job{ID: NewJobID(), Name: "fix CI", Repo: "o/r", Number: 1, PID: pid, Started: time.Now().Add(-time.Minute)}
	if err := os.MkdirAll(JobsDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(j.Path(".jsonl"), []byte(log), 0o600)
	os.WriteFile(j.Path(".err"), []byte(stderr), 0o600)
	return &JobView{Job: j}
}

// A live job's log is read incrementally, including a line split across reads.
func TestJobIncremental(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	v := writeJob(t, os.Getpid(), evInit+"\n"+evTool[:30], "")
	v.Status = JobRunning
	v.Update()
	if v.Status != JobRunning || len(v.Activity) != 0 {
		t.Fatalf("half a line: %+v", v)
	}
	f, _ := os.OpenFile(v.Path(".jsonl"), os.O_APPEND|os.O_WRONLY, 0)
	f.WriteString(evTool[30:] + "\n" + evResult + "\n")
	f.Close()
	v.Update()
	if v.Status != JobDone || len(v.Activity) != 1 {
		t.Fatalf("after rest: %+v", v)
	}
}

// End to end: a background command runs detached, gets its session id, and its stream shows up.
func TestStartJob(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	cfg := Config{Commands: []Command{{Key: "z", Name: "fake agent", Background: true,
		Run: `printf '{"type":"assistant","message":{"content":[{"type":"text","text":"%s"}]}}\n' "$PR_SESSION_ID"; sleep 0.3; echo '` + evResult + `'`}}}
	if err := validateCommands(&cfg); err != nil {
		t.Fatal(err)
	}
	pp, err := Prepare(cfg, cfg.Commands[0], PR{Repo: "o/r", Number: 3}, "")
	if err != nil {
		t.Fatal(err)
	}
	v, err := StartJob(pp)
	if err != nil {
		t.Fatal(err)
	}
	if v.Status != JobRunning {
		t.Fatalf("status = %v", v.Status)
	}
	deadline := time.Now().Add(5 * time.Second)
	for v.Status == JobRunning && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
		v.Update()
	}
	if v.Status != JobDone || len(v.Activity) != 1 || !strings.Contains(v.Activity[0], v.ID) {
		t.Fatalf("job: %+v", v)
	}

	got := LoadJobs()[PrKey("o/r", 3)]
	if got == nil || got.ID != v.ID || got.Status != JobDone {
		t.Fatalf("reloaded: %+v", got)
	}
}

// A /loop job reports a result per round and keeps running: it waits between rounds, and
// counts as active the whole time, so no second agent can start on the PR.
func TestLoopJobWaitsBetweenRounds(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	ev := `{"type":"assistant","message":{"content":[{"type":"text","text":"round"}]}}`
	c := Command{Key: "z", Name: "manage", Background: true,
		Run: "echo '" + evResult + "'; sleep 1; echo '" + ev + "'; sleep 1; echo '" + evResult + "'; sleep 1"}
	v, err := StartJob(Prepared{Cmd: c, Pr: PR{Repo: "o/r", Number: 4}, jobID: NewJobID()})
	if err != nil {
		t.Fatal(err)
	}
	var seen []string
	for end := time.Now().Add(5 * time.Second); !v.over && time.Now().Before(end); {
		time.Sleep(50 * time.Millisecond)
		v.Update()
		s := map[bool]string{true: "waiting", false: map[JobStatus]string{JobRunning: "running", JobDone: "done", JobFailed: "failed"}[v.Status]}[v.Waiting]
		if len(seen) == 0 || seen[len(seen)-1] != s {
			seen = append(seen, s)
		}
		if s != "done" && !v.Active() {
			t.Fatalf("%s but not active", s)
		}
	}
	if got := strings.Join(seen, ","); got != "waiting,running,waiting,done" {
		t.Fatalf("states: %s", got)
	}
	if v.Active() || v.Finished.IsZero() {
		t.Fatalf("finished job: %+v", v)
	}
}

// groupGone waits for every process in the job's group to exit.
func groupGone(pid int) bool {
	for i := 0; i < 100; i++ { // up to 5s: killed processes linger briefly as zombies until reaped
		if syscall.Kill(-pid, 0) == syscall.ESRCH {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func waitJob(v *JobView, d time.Duration) {
	for end := time.Now().Add(d); v.Status == JobRunning && time.Now().Before(end); {
		time.Sleep(100 * time.Millisecond)
		v.Update()
	}
}

// A job past its limit is stopped with everything it started, and says why.
func TestJobTimeLimit(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	c := Command{Key: "z", Name: "slow", Background: true, Run: "sleep 300 & sleep 300"}
	pp := Prepared{Cmd: c, Pr: PR{Repo: "o/r", Number: 1}, jobID: NewJobID()}
	pp.Cmd.limit = 2 * time.Second
	v, err := StartJob(pp)
	if err != nil {
		t.Fatal(err)
	}
	waitJob(v, 8*time.Second)
	if v.Status != JobFailed || !strings.Contains(v.Result, "time limit") {
		t.Fatalf("timed-out job: %+v", v)
	}
	if !groupGone(v.PID) {
		t.Fatal("processes from the timed-out job are still running")
	}
}

// A job that finishes on its own takes the watchdog and its leftovers with it, and never reports a timeout.
func TestJobFinishesCleanly(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	c := Command{Key: "z", Name: "quick", Background: true, Run: "sleep 300 & echo '" + evResult + "'"}
	pp := Prepared{Cmd: c, Pr: PR{Repo: "o/r", Number: 2}, jobID: NewJobID()}
	pp.Cmd.limit = 3 * time.Second
	v, err := StartJob(pp)
	if err != nil {
		t.Fatal(err)
	}
	waitJob(v, 5*time.Second)
	if v.Status != JobDone {
		t.Fatalf("job: %+v", v)
	}
	if !groupGone(v.PID) {
		t.Fatal("the watchdog or the job's leftover sleep is still running")
	}
	time.Sleep(4 * time.Second) // past the limit: no late timeout line may appear
	if b, _ := os.ReadFile(v.Path(".jsonl")); strings.Contains(string(b), "time limit") {
		t.Fatalf("finished job later reported a timeout: %s", b)
	}
}

func TestTimeoutConfig(t *testing.T) {
	cfg, err := ParseConfig([]byte("sections:\n  - title: x\n    query: is:pr\n"))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cfg.Commands {
		if (c.Key == "cf" && c.limit != 45*time.Minute) || (c.Key == "D" && c.limit != 3*time.Hour) {
			t.Fatalf("%s limit = %v", c.Key, c.limit)
		}
	}
	base := "sections:\n  - title: x\n    query: is:pr\ncommands:\n"
	for _, bad := range []string{"  - {key: z, run: x, timeout: 5m}\n", "  - {key: z, run: x, background: true, timeout: 10s}\n"} {
		if _, err := ParseConfig([]byte(base + bad)); err == nil {
			t.Errorf("%q: expected error", bad)
		}
	}
	cfg, _ = ParseConfig([]byte(base + "  - {key: z, run: x, background: true, timeout: \"0\"}\n"))
	if cfg.Commands[0].limit != 0 {
		t.Error(`timeout "0" should disable the limit`)
	}
}

// Stopping an agent ends it and what it started, and it reads as stopped, also after a reload.
func TestStopAgent(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	p := PR{Repo: "o/r", Number: 5}
	v, err := StartJob(Prepared{Cmd: Command{Key: "z", Name: "manage", Background: true, Run: "sleep 30 & sleep 30"}, Pr: p, jobID: NewJobID()})
	if err != nil {
		t.Fatal(err)
	}
	jobs := map[string]*JobView{PrKey(p.Repo, p.Number): v}
	if msg, err := StopAgent(jobs, p); err != nil || msg != "stopped manage on #5" {
		t.Fatalf("StopAgent = %q, %v", msg, err)
	}
	if !groupGone(v.PID) {
		t.Fatal("agent's processes still running")
	}
	waitJob(v, 3*time.Second)
	if !v.Stopped || v.Active() || v.Result != "stopped by you" {
		t.Fatalf("after stop: %+v", v)
	}
	if got := LoadJobs()[PrKey(p.Repo, p.Number)]; got == nil || !got.Stopped {
		t.Fatalf("reloaded: %+v", got)
	}
	if _, err := StopAgent(jobs, p); err == nil {
		t.Fatal("stopping a stopped agent should fail")
	}
}

// A reply stops a running agent first, so the session is free to resume.
func TestStopAndWait(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	v, err := StartJob(Prepared{Cmd: Command{Key: "z", Name: "manage", Background: true, Run: "sleep 30"}, Pr: PR{Repo: "o/r", Number: 6}, jobID: NewJobID()})
	if err != nil {
		t.Fatal(err)
	}
	if err := v.StopAndWait(5 * time.Second); err != nil {
		t.Fatal(err)
	}
	if v.alive() {
		t.Fatal("still running")
	}
}

// A job started by another gh-prs after this one loaded its jobs is picked up; held ones are kept.
func TestMergeJobs(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	save := func(n int, started time.Time) Job {
		j := Job{ID: NewJobID(), Name: "manage", Repo: "o/r", Number: n, PID: os.Getpid(), Started: started}
		os.MkdirAll(JobsDir(), 0o700)
		meta, _ := json.Marshal(j)
		os.WriteFile(j.Path(".json"), meta, 0o600)
		return j
	}
	old := save(1, time.Now().Add(-time.Hour))
	jobs := LoadJobs()
	held := jobs[PrKey("o/r", 1)]
	if held == nil || held.ID != old.ID {
		t.Fatalf("loaded %+v, want %s", held, old.ID)
	}
	if MergeJobs(jobs) || jobs[PrKey("o/r", 1)] != held {
		t.Fatal("nothing new on disk, but the held job was replaced")
	}
	newer := save(1, time.Now())
	other := save(2, time.Now())
	if !MergeJobs(jobs) {
		t.Fatal("new jobs not reported")
	}
	if jobs[PrKey("o/r", 1)].ID != newer.ID || jobs[PrKey("o/r", 2)].ID != other.ID {
		t.Fatalf("got %s and %v, want the jobs started elsewhere", jobs[PrKey("o/r", 1)].ID, jobs[PrKey("o/r", 2)])
	}
}
