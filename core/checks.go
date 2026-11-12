package core

// Failing checks: loading the tail of a failed GitHub Actions job's log.

import (
	"os/exec"
	"regexp"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

const LogLines = 40

var JobURLRe = regexp.MustCompile(`/actions/runs/[0-9]+/job/([0-9]+)`)

// JobLog fetches a failed Actions job's log excerpt, or an error message in its place.
func JobLog(jobID, repo string) string {
	out, err := exec.Command("gh", "run", "view", "--job", jobID, "-R", repo, "--log-failed").CombinedOutput()
	text := failureExcerpt(string(out), LogLines)
	if err != nil {
		text = "couldn't load the log: " + LastLine(string(out))
	}
	return text
}

// failureExcerpt picks up to n useful lines from `gh run view --log-failed` output: the
// ##[error] lines GitHub marks, with a little context before them, or the tail if none are
// marked. It drops the job, step and timestamp columns, debug noise and color codes.
func failureExcerpt(s string, n int) string {
	var lines []string
	firstErr, lastErr := -1, -1
	for _, l := range strings.Split(Sanitize(ansi.Strip(s)), "\n") {
		if parts := strings.SplitN(l, "\t", 3); len(parts) == 3 {
			l = parts[2]
		}
		if len(l) > 20 && l[4] == '-' && l[10] == 'T' { // 2026-11-11T20:01:02.1234567Z
			if j := strings.IndexByte(l, ' '); j > 0 && j < 40 {
				l = l[j+1:]
			}
		}
		if l == "" || strings.HasPrefix(l, "##[debug]") || strings.HasPrefix(l, "##[group]") || strings.HasPrefix(l, "##[endgroup]") {
			continue
		}
		if strings.HasPrefix(l, "##[error]") {
			l = "✗ " + strings.TrimPrefix(l, "##[error]")
			if firstErr < 0 {
				firstErr = len(lines)
			}
			lastErr = len(lines)
		}
		lines = append(lines, l)
	}
	from, to := max(len(lines)-n, 0), len(lines)
	if firstErr >= 0 {
		from, to = max(firstErr-5, 0), lastErr+1 // stop before post-job cleanup
	}
	return strings.Join(lines[from:min(from+n, to)], "\n")
}
