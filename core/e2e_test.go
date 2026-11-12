package core

// End-to-end agent flows: the built-in commands (cm, cy, ci, cx) run for real (worktree, prompt
// file, detached process, stream parsing, stopping) against a fake `claude` on PATH, which records
// how it was called. The UIs go through the same steps: AgentFor, StopAndWait, Prepare, StartJob.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type e2e struct {
	t   *testing.T
	cfg Config
	pr  PR
	log string
	wt  string // the PR's worktree
}

func newE2E(t *testing.T) *e2e {
	t.Helper()
	root, _ := filepath.EvalSymlinks(t.TempDir())
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config")) // the built-in prompts, not yours
	for k, v := range map[string]string{"GIT_CONFIG_GLOBAL": "/dev/null", "GIT_AUTHOR_NAME": "t", "GIT_AUTHOR_EMAIL": "t@t",
		"GIT_COMMITTER_NAME": "t", "GIT_COMMITTER_EMAIL": "t@t", "ANTHROPIC_API_KEY": "must-not-reach-agents"} {
		t.Setenv(k, v)
	}
	bin := filepath.Join(root, "bin")
	os.MkdirAll(bin, 0o755)
	fake, err := filepath.Abs("testdata/fake-claude")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(fake, filepath.Join(bin, "claude")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	log := filepath.Join(root, "claude-calls")
	t.Setenv("FAKE_CLAUDE_LOG", log)

	origin, repos := filepath.Join(root, "origin.git"), filepath.Join(root, "repos")
	run(t, root, "git", "init", "-q", "--bare", "-b", "main", origin)
	clone := filepath.Join(repos, "repo")
	run(t, root, "git", "clone", "-q", origin, clone)
	run(t, clone, "git", "commit", "-q", "--allow-empty", "-m", "init")
	run(t, clone, "git", "push", "-q", "origin", "HEAD:main", "HEAD:refs/heads/me/feat")

	cfg, err := ParseConfig([]byte("repos_dir: " + repos + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	return &e2e{t: t, cfg: cfg, log: log, wt: clone + "-feat",
		pr: PR{Number: 7, Repo: "o/repo", URL: "https://github.com/o/repo/pull/7", HeadRefName: "me/feat", BaseRefName: "main", Mine: true}}
}

func (e *e2e) cmd(key string) Command {
	for _, c := range e.cfg.Commands {
		if c.Key == key {
			return c
		}
	}
	e.t.Fatalf("no built-in command %q", key)
	return Command{}
}

// start runs a command the way the UIs do: check it against the PR's agent, stop that agent if
// the command replaces it, then prepare and start it.
func (e *e2e) start(c Command, current *JobView) *JobView {
	e.t.Helper()
	session, replaces, err := AgentFor(c, current)
	if err != nil {
		e.t.Fatal(err)
	}
	if replaces != nil {
		if err := replaces.StopAndWait(5 * time.Second); err != nil {
			e.t.Fatal(err)
		}
	}
	pp, err := Prepare(e.cfg, c, e.pr, session)
	if err != nil {
		e.t.Fatal(err)
	}
	v, err := StartJob(pp)
	if err != nil {
		e.t.Fatal(err)
	}
	return v
}

// until updates the job until cond holds, or fails after 10s.
func (e *e2e) until(v *JobView, what string, cond func() bool) {
	e.t.Helper()
	for end := time.Now().Add(10 * time.Second); time.Now().Before(end); time.Sleep(50 * time.Millisecond) {
		v.Update()
		if cond() {
			return
		}
	}
	e.t.Fatalf("timed out waiting for %s: %+v", what, v)
}

type call struct {
	cwd, apikey string
	args        []string
}

func (c call) has(flag, value string) bool {
	for i, a := range c.args {
		if a == flag && i+1 < len(c.args) && c.args[i+1] == value {
			return true
		}
	}
	return false
}

func (c call) prompt() string {
	for i, a := range c.args {
		if a == "-p" && i+1 < len(c.args) {
			return c.args[i+1]
		}
	}
	return ""
}

func (e *e2e) calls() []call {
	raw, _ := os.ReadFile(e.log)
	var out []call
	for _, rec := range strings.Split(strings.TrimSuffix(string(raw), "\x1e"), "\x1e") {
		f := strings.Split(strings.TrimSuffix(rec, "\x00"), "\x00")
		if len(f) < 2 {
			continue
		}
		out = append(out, call{strings.TrimPrefix(f[0], "cwd="), strings.TrimPrefix(f[1], "apikey="), f[2:]})
	}
	return out
}

// A manage loop asks a question; replying stops the loop and resumes its session with the answer,
// and the next reply resumes the same session again.
func TestE2EManageAskReply(t *testing.T) {
	e := newE2E(t)

	loop := e.start(e.cmd("cm"), nil)
	e.until(loop, "the loop to wait for its next round", func() bool { return loop.Waiting })
	sum, ok := ParseSummary(loop.Result)
	if !ok || sum.NeedsYou != "Rebase onto main?" || strings.Join(sum.Options, ",") != "Rebase,Leave it" {
		t.Fatalf("question: %+v", sum)
	}
	if _, _, err := AgentFor(e.cmd("cc"), loop); err == nil {
		t.Fatal("a second agent started while the loop runs")
	}

	reply := e.start(e.cmd("cy").WithInput("Rebase"), loop)
	if !groupGone(loop.PID) {
		t.Fatal("the loop still runs after the reply")
	}
	e.until(loop, "the loop to read as stopped", func() bool { return loop.Stopped })
	e.until(reply, "the reply to finish", func() bool { return !reply.Active() })
	if reply.Status != JobDone || reply.ID == loop.ID || reply.SessionID() != loop.ID {
		t.Fatalf("reply job: %+v (loop session %s)", reply, loop.ID)
	}

	again := e.start(e.cmd("cy").WithInput("Thanks, now rerun the tests"), reply)
	e.until(again, "the second reply to finish", func() bool { return !again.Active() })

	calls := e.calls()
	if len(calls) != 3 {
		t.Fatalf("%d claude calls, want 3", len(calls))
	}
	first, second, third := calls[0], calls[1], calls[2]
	if !first.has("--session-id", loop.ID) || !strings.HasPrefix(first.prompt(), "/loop ") || !strings.Contains(first.prompt(), "You manage this PR") {
		t.Errorf("loop call: %q", first.args)
	}
	for i, c := range []call{second, third} {
		if !c.has("--resume", loop.ID) {
			t.Errorf("reply %d doesn't resume the loop's session: %q", i+1, c.args)
		}
	}
	if p := second.prompt(); !strings.Contains(p, "\nRebase\n") || !strings.Contains(p, "carry on afterwards") {
		t.Errorf("reply prompt: %q", p)
	}
	for i, c := range calls {
		if c.cwd != e.wt || c.apikey != "" {
			t.Errorf("call %d: ran in %s with API key %q (want %s, none)", i, c.cwd, c.apikey, e.wt)
		}
	}
	if got := LoadJobs()[PrKey(e.pr.Repo, e.pr.Number)]; got == nil || got.ID != again.ID || got.SessionID() != loop.ID {
		t.Fatalf("after a reload, the PR's job is %+v", got)
	}
}

// Asking starts a new session with the question in the prompt; replying needs an agent first.
func TestE2EAsk(t *testing.T) {
	e := newE2E(t)
	if _, err := Prepare(e.cfg, e.cmd("cy").WithInput("hi"), e.pr, ""); err == nil {
		t.Fatal("a reply with no agent to reply to should fail")
	}
	ask := e.start(e.cmd("ci").WithInput("Why is CI red?"), nil)
	e.until(ask, "the answer", func() bool { return !ask.Active() })
	c := e.calls()
	if len(c) != 1 || !c[0].has("--session-id", ask.ID) || !strings.Contains(c[0].prompt(), "Why is CI red?") {
		t.Fatalf("ask call: %+v", c)
	}
	for _, a := range c[0].args {
		if a == "--resume" {
			t.Fatal("asking resumed a session")
		}
	}
}

// cx stops a manage loop and everything it started, and it stays stopped after a reload.
func TestE2EStopManage(t *testing.T) {
	e := newE2E(t)
	loop := e.start(e.cmd("cm"), nil)
	e.until(loop, "the loop to wait for its next round", func() bool { return loop.Waiting })
	jobs := map[string]*JobView{PrKey(e.pr.Repo, e.pr.Number): loop}
	if e.cmd("cx").Builtin != BuiltinStopAgent {
		t.Fatal("cx isn't the stop builtin")
	}
	if _, err := StopAgent(jobs, e.pr); err != nil {
		t.Fatal(err)
	}
	if !groupGone(loop.PID) {
		t.Fatal("the loop's processes still run")
	}
	e.until(loop, "stopped", func() bool { return loop.Stopped })
	if got := LoadJobs()[PrKey(e.pr.Repo, e.pr.Number)]; got == nil || !got.Stopped || got.Active() {
		t.Fatalf("after a reload: %+v", got)
	}
}
