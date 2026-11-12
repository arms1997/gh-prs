package main

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"gh-prs/core"
)

func TestParseSummary(t *testing.T) {
	result := "I looked at the failing check and the logs.\n\n" +
		"**DONE:** Fixed the flaky registry test\n" +
		"- `registry.test.ts`: await the tool sync\n" +
		"- `registry.ts`: guard empty palettes\n" +
		"PUSHED: a1b2c3d\n" +
		"NEEDS YOU: nothing\n"
	s, ok := parseSummary(result)
	if !ok || s.Headline != "Fixed the flaky registry test" || len(s.Changes) != 2 || s.Pushed != "a1b2c3d" || s.NeedsYou != "nothing" {
		t.Fatalf("parsed %+v ok=%v", s, ok)
	}
	if _, ok := parseSummary("just prose, no block"); ok {
		t.Fatal("prose without DONE: should not parse")
	}
}

// The agent box wraps to the pane width whatever the agent writes; height grows, and the pane scrolls.
func TestAgentBoxFitsWidth(t *testing.T) {
	long := strings.Repeat("very long words ", 40)
	var changes strings.Builder
	for i := 0; i < 12; i++ {
		changes.WriteString("- `file" + strings.Repeat("x", i) + ".ts`: " + long + "\n")
	}
	cases := map[string]*core.JobView{
		"summary":  {Job: core.Job{Name: "fix CI"}, Status: core.JobDone, Turns: 3, Cost: 0.4, Result: "DONE: " + long + "\n" + changes.String() + "PUSHED: NO: " + long + "\nNEEDS YOU: " + long},
		"prose":    {Job: core.Job{Name: "fix CI"}, Status: core.JobFailed, Result: strings.Repeat(long+"\n", 30)},
		"running":  {Job: core.Job{Name: "fix CI", Started: time.Now()}, Status: core.JobRunning, Activity: []string{long, long, long, long, long, long}},
		"no lines": {Job: core.Job{Name: "fix CI"}, Status: core.JobFailed},
	}
	for name, v := range cases {
		for _, l := range strings.Split(ansi.Strip(renderJob(v, 50)), "\n") {
			if w := lipgloss.Width(l); w > 50 {
				t.Errorf("%s: line %d wide: %q", name, w, l)
			}
		}
	}
	box := ansi.Strip(renderJob(cases["summary"], 50))
	for _, want := range []string{"+ file.ts: very long", "\n│   words", "…and 2 more", "not pushed", "! needs you"} {
		if !strings.Contains(box, want) {
			t.Errorf("summary box missing %q:\n%s", want, box)
		}
	}
	if strings.Contains(box, "…\n") || strings.Contains(box, "… │") {
		t.Errorf("summary lines should wrap, not be cut:\n%s", box)
	}
}
