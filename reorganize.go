package main

// pc: ask Claude to reorganize. Claude runs with no tools at all and only proposes text: new
// versions of config.yml and organize.yml. Only grouping may change, the result must parse,
// and nothing is written until you press y on the diff.

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"gh-prs/core"
)

type proposalMsg struct {
	p   *core.Proposal
	err error
}

func (m *model) askClaude(ask string) tea.Cmd {
	if ask == "" {
		return nil
	}
	oldCfg, _ := os.ReadFile(core.ConfigPath())
	oldOrg, _ := m.org.Marshal()
	org := *m.org
	prompt := core.OrganizePrompt(ask, oldCfg, oldOrg, m.data, m.org)
	m.proposal = &core.Proposal{Loading: true, Ask: ask}
	m.detail.SetContent(m.renderDetail())
	m.detail.GotoTop()
	return func() tea.Msg {
		res, cost, err := core.RunClaude(prompt)
		if err != nil {
			return proposalMsg{err: err}
		}
		p, err := core.BuildProposal(ask, core.Sanitize(res), oldCfg, oldOrg, &org)
		if p != nil {
			p.Cost = cost
		}
		return proposalMsg{p, err}
	}
}

func (m *model) applyProposal() tea.Cmd {
	p := m.proposal
	m.proposal = nil
	if p.Unchanged {
		m.status, m.isErr = "nothing to apply", false
		m.clamp()
		return nil
	}
	if p.Config != nil {
		cfg, err := core.ApplyConfig(p.Config)
		if err != nil { // validated before the preview, so this means the file changed underneath us
			m.status, m.isErr = err.Error(), true
			return nil
		}
		m.cfg = cfg
	}
	m.org = p.Org
	m.saveOrganize("applied Claude's reorganization" + map[bool]string{true: " (old config in config.yml.bak)", false: ""}[p.Config != nil])
	if p.Config != nil && !m.loading { // sections may have new queries
		m.loading = true
		return fetchCmd(m.cfg, m.data)
	}
	return nil
}

// renderProposal shows what Claude is doing or proposing, in the detail pane.
func renderProposal(p *core.Proposal) string {
	var b strings.Builder
	if p.Loading {
		b.WriteString(yellow.Bold(true).Render("Asking Claude to reorganize…") + "\n" + dim.Render("“"+p.Ask+"”") + "\n\n" + dim.Render("esc cancels"))
		return b.String()
	}
	b.WriteString(bold.Render("Claude's proposal") + dim.Render(fmt.Sprintf("  $%.2f", p.Cost)) + "\n")
	b.WriteString(dim.Render("“"+p.Ask+"”") + "\n")
	if p.Summary != "" {
		b.WriteString(p.Summary + "\n")
	}
	b.WriteString("\n")
	if p.Unchanged {
		b.WriteString(dim.Render("No changes proposed.") + "\n")
	}
	for _, l := range p.Diff {
		switch {
		case strings.HasPrefix(l, "## "):
			b.WriteString(bold.Render(strings.TrimPrefix(l, "## ")) + "\n")
		case strings.HasPrefix(l, "+ "):
			b.WriteString(green.Render(l) + "\n")
		case strings.HasPrefix(l, "- "):
			b.WriteString(red.Render(l) + "\n")
		default:
			b.WriteString(dim.Render(l) + "\n")
		}
	}
	b.WriteString("\n" + bold.Render("y apply · n discard"))
	return b.String()
}
