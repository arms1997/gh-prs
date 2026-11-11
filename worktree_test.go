package main

import (
	"os/exec"
	"path/filepath"
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
}
