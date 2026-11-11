package main

// The PR list: rows, columns, badges and the agent column.

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Widths of the right-aligned table columns.
const (
	colLines   = 13
	colCI      = 8
	colReview  = 11
	colThreads = 6
	colMerge   = 10
	colAgent   = 18
)

var spinner = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// shaded reports whether row i is an every-other PR row in its group, restarting after each header.
func (m model) shaded(i int) bool {
	n := 0
	for ; i >= 0 && m.rows[i].PR != nil; i-- {
		n++
	}
	return n > 0 && n%2 == 0
}

type seg struct {
	text  string
	style lipgloss.Style
}

func spaces(n int) string { return strings.Repeat(" ", max(n, 0)) }

// compactCount keeps the lines column a fixed width: 47981 → "48k". The detail pane shows exact counts.
func compactCount(n int) string {
	if n < 10000 {
		return fmt.Sprint(n)
	}
	return fmt.Sprintf("%dk", (n+500)/1000)
}

func columnHeader(w int) string {
	right := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %-*s  %-*s", colLines, "lines", colCI-1, "ci", colReview-1, "review", colThreads-1, "unres", colMerge-1, "merge", colAgent-1, "agent")
	return dim.Underline(true).Render(" pr" + spaces(w-3-lipgloss.Width(right)) + right)
}

// agentCell is the agent column: a spinner while running, then the outcome.
func agentCell(v *jobView, spin int) (string, lipgloss.Style) {
	if v == nil {
		return "", dim
	}
	switch v.Status {
	case JobRunning:
		return fmt.Sprintf(" %s %s %s", spinner[spin%len(spinner)], v.Name, shortDuration(v.elapsed())), yellow
	case JobDone:
		return " ✓ " + v.Name, green
	}
	return " ✗ " + v.Name + " ", badgeFail
}

// prColumns renders the right-aligned table cells for a PR.
func prColumns(p PR, job *jobView, spin int) []seg {
	var out []seg
	col := func(text string, st lipgloss.Style, w int) {
		out = append(out, seg{" ", plain}, seg{text, st}, seg{spaces(w - lipgloss.Width(text)), plain})
	}

	add, del := "+"+compactCount(p.Additions), "−"+compactCount(p.Deletions)
	out = append(out, seg{add, green}, seg{" ", plain}, seg{del, red}, seg{spaces(colLines - lipgloss.Width(add) - 1 - lipgloss.Width(del)), plain})

	switch p.CI {
	case "FAILURE", "ERROR":
		col(" ✗ fail ", badgeFail, colCI)
	case "PENDING", "EXPECTED":
		col(" • run", yellow, colCI)
	case "SUCCESS":
		col(" ✓ pass", green, colCI)
	default:
		col(" –", dim, colCI)
	}

	switch p.ReviewDecision {
	case "CHANGES_REQUESTED":
		col(" changes ", badgeChanges, colReview)
	case "APPROVED":
		col(" ✓ approved", green, colReview)
	case "REVIEW_REQUIRED":
		col(" waiting", dim, colReview)
	default:
		col(" –", dim, colReview)
	}

	if p.Unresolved > 0 {
		col(fmt.Sprintf(" %d", p.Unresolved), yellow, colThreads)
	} else {
		col("", dim, colThreads)
	}

	if p.Mergeable == "CONFLICTING" {
		col(" conflict ", badgeConflict, colMerge)
	} else {
		col("", dim, colMerge)
	}

	text, st := agentCell(job, spin)
	col(ansi.Truncate(text, colAgent, "…"), st, colAgent)
	return out
}

func (m model) renderRow(r Row, isSel, shade bool, w int) string {
	indent := strings.Repeat("  ", r.Depth)
	var lead []seg // markers before the PR number: whose turn, and changed since you looked
	var left seg
	var right []seg

	if r.PR == nil {
		arrow := "▾"
		if m.collapsed[r.Path] {
			arrow = "▸"
		}
		st, ok := statusStyles[r.Title]
		if !ok {
			st = bold
		}
		if r.Depth == 0 {
			st = sectionStyle
		}
		left = seg{fmt.Sprintf("%s%s %s (%d)", indent, arrow, r.Title, r.Count), st}
	} else {
		p := r.PR
		stack, on := "", ""
		if r.Stack > 0 {
			stack = strings.Repeat("  ", r.Stack-1) + "↳ "
		} else if r.StackedOn > 0 { // parent is in another group
			stack, on = "↳ ", fmt.Sprintf("(on #%d) ", r.StackedOn)
		}
		turn, ts := "○", dim
		if mine, _ := turnOf(*p); mine {
			turn, ts = "●", yellow.Bold(true)
		}
		changed := " "
		if m.isNew(*p) {
			changed = "◆"
		}
		lead = []seg{{indent, plain}, {turn, ts}, {changed, cyan}, {" ", plain}}
		left = seg{fmt.Sprintf("%s%-7s%s%s", stack, fmt.Sprintf("#%d", p.Number), on, p.Title), plain}
		if isSel {
			left.style = bold
		}
		right = prColumns(*p, m.jobs[prKey(p.Repo, p.Number)], m.spin)
	}

	leadW, rightW := 0, 0
	for _, s := range lead {
		leadW += lipgloss.Width(s.text)
	}
	for _, s := range right {
		rightW += lipgloss.Width(s.text)
	}
	left.text = ansi.Truncate(left.text, max(w-leadW-rightW-1, 0), "…")
	segs := append(lead, left, seg{spaces(w - leadW - lipgloss.Width(left.text) - rightW), plain})
	segs = append(segs, right...)

	var b strings.Builder
	for _, s := range segs {
		st := s.style
		if _, none := st.GetBackground().(lipgloss.NoColor); none { // badges keep their own color
			if isSel {
				st = st.Background(selBg)
			} else if shade {
				st = st.Background(stripeBg)
			}
		}
		b.WriteString(st.Render(s.text))
	}
	return b.String()
}
