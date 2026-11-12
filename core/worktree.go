package core

// Finding or creating the git worktree for a PR branch.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

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

// uncommitted counts changed, staged and untracked files in a worktree.
func uncommitted(dir string) (int, error) {
	out, err := git(dir, "status", "--porcelain")
	if err != nil || out == "" {
		return 0, err
	}
	return len(strings.Split(out, "\n")), nil
}
