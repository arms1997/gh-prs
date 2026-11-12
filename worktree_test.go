package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	c := exec.Command(args[0], args[1:]...)
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("%v: %v\n%s", args, err, out)
	}
}

func TestEnsureWorktree(t *testing.T) {
	root, _ := filepath.EvalSymlinks(t.TempDir()) // git reports resolved paths (/var → /private/var on macOS)
	origin, repos := filepath.Join(root, "origin.git"), filepath.Join(root, "repos")
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_AUTHOR_NAME", "t")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@t")
	t.Setenv("GIT_COMMITTER_NAME", "t")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@t")

	run(t, root, "git", "init", "-q", "--bare", "-b", "main", origin)
	clone := filepath.Join(repos, "repo")
	run(t, root, "git", "clone", "-q", origin, clone)
	run(t, clone, "git", "commit", "-q", "--allow-empty", "-m", "init")
	run(t, clone, "git", "push", "-q", "origin", "HEAD:main")
	run(t, clone, "git", "push", "-q", "origin", "HEAD:refs/heads/me/feat-x")

	cfg := Config{ReposDir: repos}
	p := PR{Number: 1, Repo: "o/repo", HeadRefName: "me/feat-x"}

	dir, err := ensureWorktree(cfg, p)
	if err != nil {
		t.Fatal(err)
	}
	if dir != clone+"-feat-x" {
		t.Fatalf("created %s", dir)
	}
	again, err := ensureWorktree(cfg, p)
	if err != nil || again != dir {
		t.Fatalf("second call: %s, %v (want reuse of %s)", again, err, dir)
	}
	run(t, clone, "git", "push", "-q", "origin", "HEAD:refs/heads/other/feat-x")
	clash, err := ensureWorktree(cfg, PR{Number: 5, Repo: "o/repo", HeadRefName: "other/feat-x"})
	if err != nil || clash != clone+"-feat-x-pr5" {
		t.Fatalf("name clash: %s, %v", clash, err)
	}
	main, err := ensureWorktree(cfg, PR{Repo: "o/repo", HeadRefName: "main"})
	if err != nil || main != clone {
		t.Fatalf("main: %s, %v (want the clone itself)", main, err)
	}
	if _, err := ensureWorktree(cfg, PR{Repo: "o/missing", HeadRefName: "x"}); err == nil {
		t.Fatal("missing clone should fail")
	}

	// Agents refuse a worktree with your uncommitted work in it; terminal commands don't care.
	agent := Command{Key: "z", Name: "agent", Background: true, Worktree: true, Run: "true"}
	if _, err := prepare(cfg, agent, p, ""); err != nil {
		t.Fatalf("clean worktree: %v", err)
	}
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("mine"), 0o644)
	if _, err := prepare(cfg, agent, p, ""); err == nil || !strings.Contains(err.Error(), "1 uncommitted change") {
		t.Fatalf("dirty worktree should be refused: %v", err)
	}
	os.Remove(filepath.Join(dir, "notes.txt"))

	// Agents push to your PRs only: on someone else's, git push fails.
	push := func(mine bool) error {
		p.Mine = mine
		pp, err := prepare(cfg, agent, p, "")
		if err != nil {
			t.Fatal(err)
		}
		c := exec.Command("git", "push", "-q", "origin", "HEAD")
		c.Dir, c.Env = dir, append(os.Environ(), pp.env...)
		return c.Run()
	}
	if err := push(false); err == nil {
		t.Fatal("push to someone else's PR should fail")
	}
	if err := push(true); err != nil {
		t.Fatalf("push to your own PR: %v", err)
	}

	shell := Command{Key: "w", Name: "shell", Terminal: true, Worktree: true, Opener: "true"}
	if _, err := prepare(cfg, shell, p, ""); err != nil {
		t.Fatalf("terminal commands should still open: %v", err)
	}
}
