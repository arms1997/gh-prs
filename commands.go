package main

// Commands: config validation, prompts, and preparing a command for a PR.

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
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
	Prompt      string `yaml:"prompt"`        // Go template, written to $PR_PROMPT_FILE after the prompt context
	PromptFile  string `yaml:"prompt_file"`   // or the same, read from a file (see readPrompt)
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
	"r": true, "q": true, "?": true, "/": true, "f": true, "i": true, "enter": true, " ": true, "tab": true, "esc": true,
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
		if c.Prompt != "" && c.PromptFile != "" {
			return fmt.Errorf("command %q: set prompt or prompt_file, not both", c.Name)
		}
		if c.PromptFile != "" {
			p, err := readPrompt(c.PromptFile)
			if err != nil {
				return fmt.Errorf("command %q: %w", c.Name, err)
			}
			c.Prompt = p
		}
		if c.Prompt != "" {
			t, err := template.New(c.Name).Option("missingkey=error").Parse(cfg.promptContext + c.Prompt)
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
