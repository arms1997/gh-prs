package core

// The summary block agents end their final message with (see defaults/prompts/context.md).

import "strings"

// AgentSummary is the block agents are asked to end with (see prompt_context).
type AgentSummary struct {
	Headline string
	Changes  []string
	Pushed   string
	NeedsYou string
}

// ParseSummary reads the last DONE: block of an agent's final message; ok is false if there is none.
func ParseSummary(result string) (AgentSummary, bool) {
	var s AgentSummary
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
