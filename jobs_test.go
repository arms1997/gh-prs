package main

import (
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

func writeJob(t *testing.T, pid int, log, stderr string) *jobView {
	t.Helper()
	j := Job{ID: newJobID(), Name: "fix CI", Repo: "o/r", Number: 1, PID: pid, Started: time.Now().Add(-time.Minute)}
	if err := os.MkdirAll(jobsDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(j.path(".jsonl"), []byte(log), 0o600)
	os.WriteFile(j.path(".err"), []byte(stderr), 0o600)
	return &jobView{Job: j}
}

// A live job's log is read incrementally, including a line split across reads.
func TestJobIncremental(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	v := writeJob(t, os.Getpid(), evInit+"\n"+evTool[:30], "")
	v.Status = JobRunning
	v.update()
	if v.Status != JobRunning || len(v.Activity) != 0 {
		t.Fatalf("half a line: %+v", v)
	}
	f, _ := os.OpenFile(v.path(".jsonl"), os.O_APPEND|os.O_WRONLY, 0)
	f.WriteString(evTool[30:] + "\n" + evResult + "\n")
	f.Close()
	v.update()
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
	pp, err := prepare(cfg, cfg.Commands[0], PR{Repo: "o/r", Number: 3}, "")
	if err != nil {
		t.Fatal(err)
	}
	v, err := startJob(pp)
	if err != nil {
		t.Fatal(err)
	}
	if v.Status != JobRunning {
		t.Fatalf("status = %v", v.Status)
	}
	deadline := time.Now().Add(5 * time.Second)
	for v.Status == JobRunning && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
		v.update()
	}
	if v.Status != JobDone || len(v.Activity) != 1 || !strings.Contains(v.Activity[0], v.ID) {
		t.Fatalf("job: %+v", v)
	}

	got := loadJobs()[prKey("o/r", 3)]
	if got == nil || got.ID != v.ID || got.Status != JobDone {
		t.Fatalf("reloaded: %+v", got)
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

func waitJob(v *jobView, d time.Duration) {
	for end := time.Now().Add(d); v.Status == JobRunning && time.Now().Before(end); {
		time.Sleep(100 * time.Millisecond)
		v.update()
	}
}

// A job past its limit is stopped with everything it started, and says why.
func TestJobTimeLimit(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	c := Command{Key: "z", Name: "slow", Background: true, Run: "sleep 300 & sleep 300"}
	pp := prepared{cmd: c, pr: PR{Repo: "o/r", Number: 1}, jobID: newJobID()}
	pp.cmd.limit = 2 * time.Second
	v, err := startJob(pp)
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
	pp := prepared{cmd: c, pr: PR{Repo: "o/r", Number: 2}, jobID: newJobID()}
	pp.cmd.limit = 3 * time.Second
	v, err := startJob(pp)
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
	if b, _ := os.ReadFile(v.path(".jsonl")); strings.Contains(string(b), "time limit") {
		t.Fatalf("finished job later reported a timeout: %s", b)
	}
}

func TestTimeoutConfig(t *testing.T) {
	cfg, err := parseConfig([]byte("sections:\n  - title: x\n    query: is:pr\n"))
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
		if _, err := parseConfig([]byte(base + bad)); err == nil {
			t.Errorf("%q: expected error", bad)
		}
	}
	cfg, _ = parseConfig([]byte(base + "  - {key: z, run: x, background: true, timeout: \"0\"}\n"))
	if cfg.Commands[0].limit != 0 {
		t.Error(`timeout "0" should disable the limit`)
	}
}

// When an agent finishes, PR data is refetched so the detail pane shows what it changed.
func TestRefreshWhenAgentFinishes(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	v := writeJob(t, os.Getpid(), evInit+"\n", "")
	v.Status = JobRunning
	m := model{jobs: map[string]*jobView{prKey(v.Repo, v.Number): v}, ticking: true, spin: 6}

	tm, cmd := m.Update(jobTickMsg{}) // spin 7: no log read yet
	if m = tm.(model); m.loading || cmd == nil {
		t.Fatalf("still running: loading=%v cmd=%v", m.loading, cmd != nil)
	}

	f, _ := os.OpenFile(v.path(".jsonl"), os.O_APPEND|os.O_WRONLY, 0)
	f.WriteString(evResult + "\n")
	f.Close()
	tm, cmd = m.Update(jobTickMsg{}) // spin 8: reads the result
	m = tm.(model)
	if v.Status != JobDone || !m.loading || cmd == nil || m.ticking {
		t.Fatalf("after finish: status=%v loading=%v cmd=%v ticking=%v", v.Status, m.loading, cmd != nil, m.ticking)
	}
}

func TestOneAgentPerPR(t *testing.T) {
	cfg, err := parseConfig([]byte("sections:\n  - title: x\n    query: is:pr\n"))
	if err != nil {
		t.Fatal(err)
	}
	var cf, ct Command
	for _, c := range cfg.Commands {
		switch c.Key {
		case "cf":
			cf = c
		case "ct":
			ct = c
		}
	}
	if !cf.Background || !ct.Session || !strings.Contains(cf.Run, `--session-id "$PR_SESSION_ID"`) || !strings.HasPrefix(cf.Run, "env -u ANTHROPIC_API_KEY claude") {
		t.Fatalf("default agent commands: cf=%+v ct=%+v", cf, ct)
	}

	p := PR{Repo: "o/r", Number: 5}
	m := model{cfg: cfg, jobs: map[string]*jobView{prKey("o/r", 5): {Job: Job{ID: "x", Name: "rebase"}, Status: JobRunning}}}
	if m.startCommand(cf, p) != nil || !m.isErr {
		t.Fatal("a second agent on the same PR should be refused")
	}
	if m.startCommand(ct, p) != nil || !m.isErr {
		t.Fatal("taking over a running session should be refused")
	}
	if _, err := prepare(cfg, ct, p, ""); err == nil {
		t.Fatal("ct without a previous agent run should fail")
	}
	if _, err := parseConfig([]byte("sections:\n  - title: x\n    query: is:pr\ncommands:\n  - {key: z, run: x, background: true, terminal: true}\n")); err == nil {
		t.Fatal("background and terminal together should fail")
	}
}
