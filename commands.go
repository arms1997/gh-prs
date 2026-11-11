package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"
)

// Command is a user-defined action bound to a key sequence. PR data reaches the shell only
// through $PR_* environment variables, never by string interpolation, because branch names
// and titles on other people's PRs are untrusted.
type Command struct {
	Key         string `yaml:"key"`
	Name        string `yaml:"name"`
	Run         string `yaml:"run"`           // sh command; may be empty for terminal commands (just opens a shell)
	Prompt      string `yaml:"prompt"`        // Go template, written to $PR_PROMPT_FILE
	Worktree    bool   `yaml:"worktree"`      // run inside the PR branch's worktree, creating it if needed
	Terminal    bool   `yaml:"terminal"`      // hand the command to an opener instead of running it here
	Opener      string `yaml:"opener"`        // overrides Config.Opener for this command
	Interactive bool   `yaml:"interactive"`   // suspend the TUI and give the command the terminal
	Background  bool   `yaml:"background"`    // run detached as an agent job; stdout must be Claude stream-json
	Session     bool   `yaml:"agent_session"` // needs the PR's last agent job; sets $PR_AGENT_SESSION
	Timeout     string `yaml:"timeout"`       // background only: stop the job after this long (default 45m, "0" = never)
	Confirm     bool   `yaml:"confirm"`       // ask y/N first
	Refresh     bool   `yaml:"refresh"`       // refresh PRs after it succeeds

	tmpl  *template.Template
	limit time.Duration // parsed Timeout
}

const defaultJobLimit = 45 * time.Minute

// builtinKeys can't start a command key sequence.
var builtinKeys = map[string]bool{
	"j": true, "k": true, "g": true, "G": true, "h": true, "l": true, "e": true, "o": true,
	"r": true, "q": true, "?": true, "/": true, "enter": true, " ": true, "tab": true, "esc": true,
	"up": true, "down": true, "left": true, "right": true,
}

func validateCommands(cfg *Config) error {
	for i := range cfg.Commands {
		c := &cfg.Commands[i]
		if c.Key == "" {
			return fmt.Errorf("command %d has no key", i+1)
		}
		if c.Name == "" {
			c.Name = c.Key
		}
		if builtinKeys[string(c.Key[0])] {
			return fmt.Errorf("command %q: key %q starts with a built-in key", c.Name, c.Key)
		}
		for _, o := range cfg.Commands[:i] {
			if strings.HasPrefix(o.Key, c.Key) || strings.HasPrefix(c.Key, o.Key) {
				return fmt.Errorf("commands %q and %q: keys %q and %q overlap", o.Name, c.Name, o.Key, c.Key)
			}
		}
		if c.Run == "" && !c.Terminal {
			return fmt.Errorf("command %q has nothing to run", c.Name)
		}
		if c.Opener == "" {
			c.Opener = cfg.Opener
		}
		if c.Terminal && c.Opener == "" {
			return fmt.Errorf("command %q needs an `opener`", c.Name)
		}
		if n := btoi(c.Terminal) + btoi(c.Interactive) + btoi(c.Background); n > 1 {
			return fmt.Errorf("command %q: pick one of terminal, interactive, background", c.Name)
		}
		if c.Background && c.Run == "" {
			return fmt.Errorf("command %q: background needs a run command", c.Name)
		}
		if c.Timeout != "" && !c.Background {
			return fmt.Errorf("command %q: timeout only applies to background commands", c.Name)
		}
		if c.Background {
			c.limit = defaultJobLimit
			if c.Timeout == "0" {
				c.limit = 0
			} else if c.Timeout != "" {
				d, err := time.ParseDuration(c.Timeout)
				if err != nil || d < time.Minute {
					return fmt.Errorf("command %q: timeout %q: use a duration of at least 1m, or 0 for none", c.Name, c.Timeout)
				}
				c.limit = d
			}
		}
		if c.Prompt != "" {
			t, err := template.New(c.Name).Option("missingkey=error").Parse(cfg.PromptContext + c.Prompt)
			if err != nil {
				return fmt.Errorf("command %q prompt: %w", c.Name, err)
			}
			c.tmpl = t
		}
	}
	return nil
}

// promptData is what prompt templates can use. The title is left out on purpose: it's free text
// from the PR author and would go straight into Claude's instructions.
type promptData struct {
	Number int
	Repo   string
	URL    string
	Head   string
	Base   string
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// prepared is a command ready to execute: env and working directory resolved, files written.
type prepared struct {
	cmd    Command
	pr     PR
	env    []string
	dir    string
	script string // launcher script for terminal commands
	jobID  string // background commands: the job and Claude session id
}

// prepare resolves a command for a PR. agentSession is the PR's last agent job, for agent_session commands.
func prepare(cfg Config, c Command, p PR, agentSession string) (prepared, error) {
	pp := prepared{cmd: c, pr: p}
	if c.Background {
		pp.jobID = newJobID()
	}
	vars := [][2]string{
		{"PR_NUMBER", fmt.Sprint(p.Number)},
		{"PR_REPO", p.Repo},
		{"PR_URL", p.URL},
		{"PR_HEAD", p.HeadRefName},
		{"PR_BASE", p.BaseRefName},
		{"PR_KEY", c.Key},
		{"PR_ACTION", c.Name},
		{"GH_PRS_DIR", installDir()},
	}
	if pp.jobID != "" {
		vars = append(vars, [2]string{"PR_SESSION_ID", pp.jobID})
	}
	if c.Session {
		if agentSession == "" {
			return pp, fmt.Errorf("no agent has run on this PR yet")
		}
		vars = append(vars, [2]string{"PR_AGENT_SESSION", agentSession})
	}
	if c.Worktree {
		dir, err := ensureWorktree(cfg, p)
		if err != nil {
			return pp, err
		}
		pp.dir = dir
		vars = append(vars, [2]string{"PR_WORKTREE", dir})
	}
	if c.tmpl != nil {
		var buf bytes.Buffer
		if err := c.tmpl.Execute(&buf, promptData{p.Number, p.Repo, p.URL, p.HeadRefName, p.BaseRefName}); err != nil {
			return pp, err
		}
		f, err := promptFile(pp.jobID)
		if err != nil {
			return pp, err
		}
		_, err = f.Write(buf.Bytes())
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return pp, err
		}
		vars = append(vars, [2]string{"PR_PROMPT_FILE", f.Name()})
	}
	if c.Terminal {
		s, err := writeLauncher(vars, pp.dir, c.Run)
		if err != nil {
			return pp, err
		}
		pp.script = s
		vars = append(vars, [2]string{"PR_LAUNCH", s})
	}
	for _, v := range vars {
		pp.env = append(pp.env, v[0]+"="+v[1])
	}
	return pp, nil
}

// installDir is where the gh-prs binary lives, so commands can use the scripts/ and openers/ shipped beside it.
func installDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	return filepath.Dir(exe)
}

// promptFile is a temp file, or for a background job a file kept beside its log as a record of what it was asked.
func promptFile(jobID string) (*os.File, error) {
	if jobID == "" {
		return os.CreateTemp("", "gh-prs-prompt-*.md") // 0600
	}
	if err := os.MkdirAll(jobsDir(), 0o700); err != nil {
		return nil, err
	}
	return os.OpenFile(filepath.Join(jobsDir(), jobID+".prompt.md"), os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
}

// shQuote single-quotes s for POSIX sh.
func shQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// launcherScript is run by whatever the opener starts (a terminal app, a multiplexer), which won't
// inherit our environment, so values are baked in quoted.
func launcherScript(vars [][2]string, dir, run string) string {
	var b strings.Builder
	b.WriteString("#!/bin/sh\nrm -f -- \"$0\"\n")
	for _, v := range vars {
		fmt.Fprintf(&b, "export %s=%s\n", v[0], shQuote(v[1]))
	}
	if dir != "" {
		fmt.Fprintf(&b, "cd %s || exit 1\n", shQuote(dir))
	}
	if run != "" {
		b.WriteString(run + "\n")
	}
	b.WriteString("[ -n \"$PR_PROMPT_FILE\" ] && rm -f -- \"$PR_PROMPT_FILE\"\n")
	b.WriteString("exec \"${SHELL:-/bin/sh}\" -l\n") // leave a shell open in the worktree when the command exits
	return b.String()
}

func writeLauncher(vars [][2]string, dir, run string) (string, error) {
	f, err := os.CreateTemp("", "gh-prs-launch-*.sh")
	if err != nil {
		return "", err
	}
	_, err = f.WriteString(launcherScript(vars, dir, run))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(f.Name(), 0o700)
	}
	return f.Name(), err
}

// execCmd builds the process for a prepared command.
func (pp prepared) execCmd() *exec.Cmd {
	var c *exec.Cmd
	if pp.cmd.Terminal {
		// The script path goes in as $1 (and $PR_LAUNCH), so nothing is interpolated into the opener.
		c = exec.Command("sh", "-c", pp.cmd.Opener+` "$1"`, "sh", pp.script)
		c.Dir = pp.dir
	} else if pp.cmd.Background && pp.cmd.limit > 0 {
		c = exec.Command("sh", "-c", withTimeLimit(pp.cmd.Run, pp.cmd.limit))
		c.Dir = pp.dir
	} else {
		c = exec.Command("sh", "-c", pp.cmd.Run)
		c.Dir = pp.dir
	}
	c.Env = append(os.Environ(), pp.env...)
	return c
}

// withTimeLimit wraps a background job's command. The job leads its own process group (Setsid),
// so a watchdog in the group can stop everything the job started, even after gh-prs has quit.
// It writes a result event first so gh-prs shows why the job ended. When the command finishes
// on its own, the whole group is killed: the watchdog (which ignores TERM, as does its sleep)
// and anything the job left running. The exit code doesn't matter: status comes from the log.
func withTimeLimit(run string, limit time.Duration) string {
	result, _ := json.Marshal(map[string]any{
		"type": "result", "subtype": "error_timeout", "is_error": true,
		"result": "Stopped: hit the " + shortDuration(limit) + " time limit.",
	})
	return fmt.Sprintf(`(
%s
) &
job=$!
( trap '' TERM; sleep %d; printf '\n%%s\n' %s; kill -TERM -$$
  i=0; while kill -0 $job 2>/dev/null && [ $i -lt 10 ]; do sleep 1; i=$((i+1)); done
  kill -KILL -$$ ) &
wait $job
kill -KILL -$$`, run, int(limit.Seconds()), shQuote(string(result)))
}

var unsafePathChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[1:])
	}
	return p
}

func git(dir string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %s", args[0], strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// ensureWorktree returns the worktree that has the PR branch checked out, creating
// <clone>-<branch-name> next to the main clone if none does.
func ensureWorktree(cfg Config, p PR) (string, error) {
	clone := filepath.Join(expandHome(cfg.ReposDir), filepath.Base(p.Repo))
	if _, err := os.Stat(filepath.Join(clone, ".git")); err != nil {
		return "", fmt.Errorf("no local clone of %s at %s (set repos_dir)", p.Repo, clone)
	}
	branch := p.HeadRefName
	list, err := git(clone, "worktree", "list", "--porcelain")
	if err != nil {
		return "", err
	}
	var path string
	for _, line := range strings.Split(list, "\n") {
		if v, ok := strings.CutPrefix(line, "worktree "); ok {
			path = v
		} else if line == "branch refs/heads/"+branch {
			return path, nil
		}
	}

	if _, err := git(clone, "fetch", "origin", branch); err != nil {
		return "", fmt.Errorf("%w (PRs from forks aren't supported yet)", err)
	}
	name := branch[strings.LastIndex(branch, "/")+1:]
	name = strings.Trim(unsafePathChars.ReplaceAllString(name, "-"), "-.")
	if name == "" {
		name = fmt.Sprintf("pr-%d", p.Number)
	}
	dir := clone + "-" + name
	if _, err := os.Stat(dir); err == nil { // taken by another branch with the same last segment
		dir = fmt.Sprintf("%s-pr%d", dir, p.Number)
		if _, err := os.Stat(dir); err == nil {
			return "", fmt.Errorf("%s already exists but isn't a worktree for %s", dir, branch)
		}
	}
	if _, err := git(clone, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
		_, err = git(clone, "worktree", "add", dir, branch)
		return dir, err
	}
	_, err = git(clone, "worktree", "add", "--track", "-b", branch, dir, "origin/"+branch)
	return dir, err
}
