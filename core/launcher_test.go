package core

import (
	"os"
	"path/filepath"
	"testing"
)

const hostile = `x';touch PWNED;'$(touch PWNED2)` + "`touch PWNED3`"

// Hostile PR data must arrive in commands verbatim and never execute.
func TestPRDataIsNotInterpolated(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	p := PR{Number: 7, Repo: "o/r", URL: "https://x/" + hostile, HeadRefName: hostile, BaseRefName: "main"}

	// Background command: env vars.
	cfg := Config{}
	pp, err := Prepare(cfg, Command{Name: "t", Run: `printf %s "$PR_HEAD"`}, p, "")
	if err != nil {
		t.Fatal(err)
	}
	out, err := pp.ExecCmd().Output()
	if err != nil || string(out) != hostile {
		t.Fatalf("background: got %q, %v", out, err)
	}

	// Terminal command: launcher script with baked-in values; the opener just runs it.
	t.Setenv("SHELL", "/usr/bin/true") // the launcher ends by exec'ing $SHELL
	tcfg := Config{Opener: "sh", Commands: []Command{{Key: "z", Name: "t", Terminal: true, Run: `printf %s "$PR_HEAD" > ` + filepath.Join(dir, "out")}}}
	if err := validateCommands(&tcfg); err != nil {
		t.Fatal(err)
	}
	pp, err = Prepare(tcfg, tcfg.Commands[0], p, "")
	if err != nil {
		t.Fatal(err)
	}
	if b, err := pp.ExecCmd().CombinedOutput(); err != nil {
		t.Fatalf("launcher: %v %s", err, b)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "out")); string(got) != hostile {
		t.Fatalf("terminal: got %q", got)
	}
	if _, err := os.Stat(pp.script); !os.IsNotExist(err) {
		t.Error("launcher script should delete itself")
	}

	for _, f := range []string{"PWNED", "PWNED2", "PWNED3"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			t.Fatalf("%s was created: PR data was executed", f)
		}
	}
}

// A custom opener gets the PR env and the launcher, and a per-command opener wins over the global one.
func TestCustomOpener(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHELL", "/usr/bin/true")
	out := filepath.Join(dir, "opener.log")
	cfg := Config{Opener: "false", Commands: []Command{{
		Key: "z", Name: "claude", Terminal: true, Run: "true",
		Opener: `printf '%s|%s|%s|' "$PR_NUMBER" "$PR_KEY" "$PR_ACTION" > ` + out + ` && [ "$1" = "$PR_LAUNCH" ] && sh "$1"`,
	}}}
	if err := validateCommands(&cfg); err != nil {
		t.Fatal(err)
	}
	pp, err := Prepare(cfg, cfg.Commands[0], PR{Number: 42}, "")
	if err != nil {
		t.Fatal(err)
	}
	if b, err := pp.ExecCmd().CombinedOutput(); err != nil {
		t.Fatalf("opener: %v %s", err, b)
	}
	if got, _ := os.ReadFile(out); string(got) != "42|z|claude|" {
		t.Fatalf("opener env = %q", got)
	}

	if _, err := ParseConfig([]byte("terminal: x\n")); err == nil {
		t.Error("unknown config keys should be rejected")
	}
}
