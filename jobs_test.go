package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"gh-prs/core"
)

func deadPID(t *testing.T) int {
	c := exec.Command("true")
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	return c.Process.Pid
}

func writeJob(t *testing.T, pid int, log, stderr string) *core.JobView {
	t.Helper()
	j := core.Job{ID: core.NewJobID(), Name: "fix CI", Repo: "o/r", Number: 1, PID: pid, Started: time.Now().Add(-time.Minute)}
	if err := os.MkdirAll(core.JobsDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(j.Path(".jsonl"), []byte(log), 0o600)
	os.WriteFile(j.Path(".err"), []byte(stderr), 0o600)
	return &core.JobView{Job: j}
}

// When an agent finishes, PR data is refetched so the detail pane shows what it changed.
func TestRefreshWhenAgentFinishes(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	exited := exec.Command("true")
	if err := exited.Run(); err != nil {
		t.Fatal(err)
	}
	v := writeJob(t, exited.Process.Pid, evInit+"\n", "") // a live process would be a /loop job waiting for its next round
	v.Status = core.JobRunning
	m := model{jobs: map[string]*core.JobView{core.PrKey(v.Repo, v.Number): v}, ticking: true, spin: 6}

	tm, cmd := m.Update(jobTickMsg{}) // spin 7: no log read yet
	if m = tm.(model); m.loading || cmd == nil {
		t.Fatalf("still running: loading=%v cmd=%v", m.loading, cmd != nil)
	}

	f, _ := os.OpenFile(v.Path(".jsonl"), os.O_APPEND|os.O_WRONLY, 0)
	f.WriteString(evResult + "\n")
	f.Close()
	tm, cmd = m.Update(jobTickMsg{}) // spin 8: reads the result
	m = tm.(model)
	if v.Status != core.JobDone || !m.loading || cmd == nil || m.ticking {
		t.Fatalf("after finish: status=%v loading=%v cmd=%v ticking=%v", v.Status, m.loading, cmd != nil, m.ticking)
	}
}

func TestOneAgentPerPR(t *testing.T) {
	cfg, err := core.ParseConfig([]byte("sections:\n  - title: x\n    query: is:pr\n"))
	if err != nil {
		t.Fatal(err)
	}
	var cf, ct core.Command
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

	p := core.PR{Repo: "o/r", Number: 5}
	m := model{cfg: cfg, jobs: map[string]*core.JobView{core.PrKey("o/r", 5): {Job: core.Job{ID: "x", Name: "rebase"}, Status: core.JobRunning}}}
	if m.startCommand(cf, p) != nil || !m.isErr {
		t.Fatal("a second agent on the same PR should be refused")
	}
	if m.startCommand(ct, p) != nil || !m.isErr {
		t.Fatal("taking over a running session should be refused")
	}
	if _, err := core.Prepare(cfg, ct, p, ""); err == nil {
		t.Fatal("ct without a previous agent run should fail")
	}
	if _, err := core.ParseConfig([]byte("sections:\n  - title: x\n    query: is:pr\ncommands:\n  - {key: z, run: x, background: true, terminal: true}\n")); err == nil {
		t.Fatal("background and terminal together should fail")
	}
}

const (
	evInit   = `{"type":"system","subtype":"init","session_id":"s"}`
	evResult = `{"type":"result","subtype":"success","is_error":false,"num_turns":7,"result":"Pushed abc123: fixed the test.","total_cost_usd":0.42}`
)
