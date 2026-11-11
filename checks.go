package main

// Failing checks: loading the tail of a failed GitHub Actions job's log into the detail pane.

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

const logLines = 40

var jobURLRe = regexp.MustCompile(`/actions/runs/[0-9]+/job/([0-9]+)`)

// loadFailLog fetches the first failing Actions job's log. Other checks (external status
// checks like Rocky) have no log to fetch, so it says so.
func (m *model) loadFailLog(p PR) tea.Cmd {
	if len(p.FailingChecks) == 0 {
		m.status, m.isErr = "no failing checks on this PR", false
		return nil
	}
	k := prKey(p.Repo, p.Number)
	for _, c := range p.FailingChecks {
		id := jobURLRe.FindStringSubmatch(c.URL)
		if id == nil {
			continue
		}
		m.logs[k] = &failLog{check: c.Name, loading: true}
		m.detail.SetContent(m.renderDetail())
		return func() tea.Msg {
			out, err := exec.Command("gh", "run", "view", "--job", id[1], "-R", p.Repo, "--log-failed").CombinedOutput()
			text := failureExcerpt(string(out), logLines)
			if err != nil {
				text = "couldn't load the log: " + lastLine(string(out))
			}
			return logMsg{k, failLog{check: c.Name, text: text}}
		}
	}
	m.status, m.isErr = fmt.Sprintf("%s isn't a GitHub Actions job, so there's no log to load; o opens the PR", p.FailingChecks[0].Name), false
	return nil
}

// failureExcerpt picks up to n useful lines from `gh run view --log-failed` output: the
// ##[error] lines GitHub marks, with a little context before them, or the tail if none are
// marked. It drops the job, step and timestamp columns, debug noise and color codes.
func failureExcerpt(s string, n int) string {
	var lines []string
	firstErr, lastErr := -1, -1
	for _, l := range strings.Split(sanitize(ansi.Strip(s)), "\n") {
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
