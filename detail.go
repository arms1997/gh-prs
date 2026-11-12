package main

// The detail pane: PR details and the agent box.

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func ciText(state string) string {
	if state == "" {
		return "no checks"
	}
	return strings.ToLower(state)
}

func reviewText(d string) string {
	switch d {
	case "APPROVED":
		return "approved"
	case "CHANGES_REQUESTED":
		return "changes requested"
	case "REVIEW_REQUIRED":
		return "review required"
	}
	return "none required"
}

// agentSummary is the block agents are asked to end with (see prompt_context).
type agentSummary struct {
	Headline string
	Changes  []string
	Pushed   string
	NeedsYou string
}

// parseSummary reads the last DONE: block of an agent's final message; ok is false if there is none.
func parseSummary(result string) (agentSummary, bool) {
	var s agentSummary
	lines := strings.Split(result, "\n")
	start := -1
	for i, l := range lines {
		if strings.HasPrefix(cleanLine(l), "DONE:") {
			start = i
		}
	}
	if start < 0 {
		return s, false
	}
	for _, l := range lines[start:] {
		l = cleanLine(l)
		switch {
		case strings.HasPrefix(l, "DONE:"):
			s.Headline = strings.TrimSpace(strings.TrimPrefix(l, "DONE:"))
		case strings.HasPrefix(l, "- "), strings.HasPrefix(l, "* "):
			s.Changes = append(s.Changes, strings.TrimSpace(l[2:]))
		case strings.HasPrefix(l, "PUSHED:"):
			s.Pushed = strings.TrimSpace(strings.TrimPrefix(l, "PUSHED:"))
		case strings.HasPrefix(l, "NEEDS YOU:"):
			s.NeedsYou = strings.TrimSpace(strings.TrimPrefix(l, "NEEDS YOU:"))
		}
	}
	return s, s.Headline != ""
}

// cleanLine drops markdown emphasis and code fences agents sometimes wrap the block in.
func cleanLine(l string) string {
	l = strings.TrimSpace(strings.ReplaceAll(l, "**", ""))
	return strings.TrimSpace(strings.TrimLeft(l, "#>"))
}

var codeSpan = regexp.MustCompile("`[^`]+`")

// colorSpans styles s, showing `code spans` (usually file names) in cyan without the backticks.
func colorSpans(s string, st lipgloss.Style) string {
	var b strings.Builder
	last := 0
	for _, loc := range codeSpan.FindAllStringIndex(s, -1) {
		b.WriteString(st.Render(s[last:loc[0]]) + cyan.Render(s[loc[0]+1:loc[1]-1]))
		last = loc[1]
	}
	b.WriteString(st.Render(s[last:]))
	return b.String()
}

// fit cuts a line to width, for one-line entries like the running agent's recent actions.
func fit(s string, width int, st lipgloss.Style) string {
	return colorSpans(ansi.Truncate(s, width, "…"), st)
}

// wrapped word-wraps s to width after prefix, indenting continuation lines under the text.
func wrapped(prefix, s string, width int, st lipgloss.Style) []string {
	pw := lipgloss.Width(prefix)
	body := lipgloss.NewStyle().Width(max(width-pw, 10)).Render(colorSpans(s, st))
	lines := strings.Split(body, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " ")
		if i == 0 {
			lines[i] = st.Render(prefix) + lines[i]
		} else {
			lines[i] = spaces(pw) + lines[i]
		}
	}
	return lines
}

const maxChanges = 10

// renderJob draws the agent box: what the agent did, wrapped to the pane width. It can grow
// taller than the pane, which scrolls (e, then j/k).
func renderJob(v *jobView, width int) string {
	inner := max(width-4, 10) // border + padding on each side
	var lines []string
	add := func(s string, st lipgloss.Style) { lines = append(lines, fit(s, inner, st)) }
	wrap := func(prefix, s string, st lipgloss.Style) { lines = append(lines, wrapped(prefix, s, inner, st)...) }

	title, border := "Agent · "+v.Name, lipgloss.Color("3")
	switch v.Status {
	case JobRunning:
		add(title+" · running "+shortDuration(v.elapsed()), yellow.Bold(true))
		acts := v.Activity
		if len(acts) > 4 {
			acts = acts[len(acts)-4:]
		}
		for _, a := range acts {
			add("› "+a, dim)
		}
		if len(acts) == 0 {
			add("starting…", dim)
		}
	case JobDone, JobFailed:
		if v.Waiting {
			add(title+" · last round ✓ · waiting for the next", cyan.Bold(true))
			border = lipgloss.Color("6")
		} else if v.Status == JobDone {
			add(title+" ✓", green.Bold(true))
			border = lipgloss.Color("2")
		} else {
			add(title+" ✗", red.Bold(true))
			border = lipgloss.Color("1")
		}
		if sum, ok := parseSummary(v.Result); ok {
			wrap("", sum.Headline, bold)
			for i, c := range sum.Changes {
				if i == maxChanges {
					add(fmt.Sprintf("  …and %d more", len(sum.Changes)-maxChanges), dim)
					break
				}
				wrap("+ ", c, green)
			}
			switch p := sum.Pushed; {
			case strings.HasPrefix(strings.ToUpper(p), "NO"):
				wrap("not pushed: ", strings.TrimSpace(strings.TrimLeft(p[2:], ":")), yellow)
			case p != "":
				wrap("↑ pushed ", p, green)
			}
			if n := sum.NeedsYou; n != "" && !strings.EqualFold(strings.Trim(n, ". "), "nothing") {
				wrap("! needs you: ", n, yellow.Bold(true))
			}
		} else { // no summary block (an older prompt, the deploy script, an error)
			for _, l := range strings.Split(strings.TrimSpace(v.Result), "\n") {
				wrap("", l, plain)
			}
		}
		meta := "after " + shortDuration(v.elapsed())
		if v.Waiting {
			meta = "running " + shortDuration(v.elapsed())
		} else if v.Status == JobDone {
			meta = "done in " + shortDuration(v.elapsed())
		}
		if v.Turns > 0 {
			meta += fmt.Sprintf(" · %d turns", v.Turns)
		}
		if v.Cost > 0 {
			meta += fmt.Sprintf(" · $%.2f", v.Cost)
		}
		wrap("", meta+" · ct take over", dim)
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(border).
		Padding(0, 1).Width(inner + 2).Render(strings.Join(lines, "\n"))
}

func (m model) renderDetail() string {
	r := m.current()
	w := max(m.detail.Width, 1)
	wrap := lipgloss.NewStyle().Width(w)
	if m.showHelp {
		return wrap.Render(m.helpText())
	}
	if m.proposal != nil {
		return wrap.Render(renderProposal(m.proposal))
	}
	if r == nil {
		return ""
	}
	if r.PR == nil {
		return wrap.Render(bold.Render(r.Title) + "\n\n" + fmt.Sprintf("%d PRs", r.Count))
	}
	p := r.PR
	var b strings.Builder
	b.WriteString(bold.Render(p.Title) + "\n")
	b.WriteString(dim.Render(fmt.Sprintf("%s#%d · @%s", p.Repo, p.Number, p.Author)) + "\n")
	branches := fmt.Sprintf("%s → %s", p.HeadRefName, p.BaseRefName)
	if r.StackedOn > 0 {
		branches += fmt.Sprintf(" (stacked on #%d)", r.StackedOn)
	}
	b.WriteString(dim.Render(branches) + "\n\n")
	fmt.Fprintf(&b, "%s %s  %d files\n", green.Render(fmt.Sprintf("+%d", p.Additions)), red.Render(fmt.Sprintf("−%d", p.Deletions)), p.ChangedFiles)
	fmt.Fprintf(&b, "%s · CI %s · review %s · %d unresolved\n", bold.Render(statusOf(*p).String()), ciText(p.CI), reviewText(p.ReviewDecision), p.Unresolved)
	if p.Mergeable == "CONFLICTING" {
		b.WriteString(red.Render("Merge conflict") + "\n")
	}
	if len(p.Labels) > 0 {
		b.WriteString(dim.Render("labels: "+strings.Join(p.Labels, ", ")) + "\n")
	}
	if mine, why := turnOf(*p); mine {
		b.WriteString(yellow.Bold(true).Render("● Your turn: "+why) + "\n")
	} else {
		b.WriteString(dim.Render("○ Their turn: "+why) + "\n")
	}
	k := prKey(p.Repo, p.Number)
	if k == m.recentKey && len(m.recent) > 0 {
		b.WriteString(cyan.Render("◆ Since you last looked: "+strings.Join(m.recent, " · ")) + "\n")
	}
	if len(p.FailingChecks) > 0 {
		names := make([]string, len(p.FailingChecks))
		for i, c := range p.FailingChecks {
			names[i] = "✗ " + c.Name
		}
		b.WriteString(red.Render("Failing: "+strings.Join(names, " · ")) + dim.Render("  (f: log)") + "\n")
	}
	if p.Linear != "" {
		link := ""
		if linearURL(m.cfg.LinearWorkspace, p.Linear) != "" {
			link = "  (i: open)"
		}
		b.WriteString(dim.Render("Linear: "+p.Linear+link) + "\n")
	}
	if l := m.logs[k]; l != nil {
		if l.loading {
			b.WriteString("\n" + dim.Render("Loading the "+l.check+" log…") + "\n")
		} else {
			b.WriteString("\n" + red.Bold(true).Render("Log · "+l.check) + dim.Render(fmt.Sprintf(" (%d lines around the errors)", logLines)) + "\n" + dim.Render(l.text) + "\n")
		}
	}
	if v := m.jobs[k]; v != nil {
		b.WriteString("\n" + renderJob(v, w) + "\n")
	}
	body := strings.TrimSpace(p.Body)
	if body == "" {
		body = dim.Render("(no description)")
	}
	b.WriteString("\n" + body)
	return wrap.Render(b.String())
}
