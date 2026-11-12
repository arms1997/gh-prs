package core

// Running commands: launcher scripts for openers, shell quoting, job time limits.

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

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
func (pp Prepared) ExecCmd() *exec.Cmd {
	var c *exec.Cmd
	if pp.Cmd.Terminal {
		// The script path goes in as $1 (and $PR_LAUNCH), so nothing is interpolated into the opener.
		c = exec.Command("sh", "-c", pp.Cmd.Opener+` "$1"`, "sh", pp.script)
		c.Dir = pp.dir
	} else if pp.Cmd.Background && pp.Cmd.limit > 0 {
		c = exec.Command("sh", "-c", withTimeLimit(pp.Cmd.Run, pp.Cmd.limit))
		c.Dir = pp.dir
	} else {
		c = exec.Command("sh", "-c", pp.Cmd.Run)
		c.Dir = pp.dir
	}
	c.Env = append(os.Environ(), pp.Env...)
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
		"result": "Stopped: hit the " + ShortDuration(limit) + " time limit.",
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
