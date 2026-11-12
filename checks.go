package main

// Failing checks: loading the tail of a failed GitHub Actions job's log into the detail pane.

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"gh-prs/core"
)

// loadFailLog fetches the first failing Actions job's log. Other checks (external status
// checks like Rocky) have no log to fetch, so it says so.
func (m *model) loadFailLog(p core.PR) tea.Cmd {
	if len(p.FailingChecks) == 0 {
		m.status, m.isErr = "no failing checks on this PR", false
		return nil
	}
	k := core.PrKey(p.Repo, p.Number)
	for _, c := range p.FailingChecks {
		id := core.JobURLRe.FindStringSubmatch(c.URL)
		if id == nil {
			continue
		}
		m.logs[k] = &failLog{check: c.Name, loading: true}
		m.detail.SetContent(m.renderDetail())
		return func() tea.Msg {
			return logMsg{k, failLog{check: c.Name, text: core.JobLog(id[1], p.Repo)}}
		}
	}
	m.status, m.isErr = fmt.Sprintf("%s isn't a GitHub Actions job, so there's no log to load; o opens the PR", p.FailingChecks[0].Name), false
	return nil
}
