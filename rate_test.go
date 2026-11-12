package main

import (
	"testing"
	"time"
)

func TestPageSize(t *testing.T) {
	prev := []sectionData{
		{Section: Section{Title: "few"}, PRs: make([]PR, 3)},
		{Section: Section{Title: "some"}, PRs: make([]PR, 17)},
		{Section: Section{Title: "lots"}, PRs: make([]PR, 95)},
	}
	for title, want := range map[string]int{"few": 13, "some": 27, "lots": 100, "new": 100} {
		if got := pageSize(prev, title); got != want {
			t.Errorf("%s: %d, want %d", title, got, want)
		}
	}
	if got := pageSize([]sectionData{{Section: Section{Title: "empty"}}}, "empty"); got != 10 {
		t.Errorf("empty section: %d, want 10", got)
	}
}

func TestRefreshDelay(t *testing.T) {
	now := time.Now()
	every := 2 * time.Minute
	reset := now.Add(40 * time.Minute)
	if d := refreshDelay(every, rateLimit{Remaining: 4000, ResetAt: reset}, now); d != every {
		t.Errorf("plenty left: %v", d)
	}
	if d := refreshDelay(every, rateLimit{Remaining: 200, ResetAt: reset}, now); d != 40*time.Minute {
		t.Errorf("low: %v, want to wait for the reset", d)
	}
	if d := refreshDelay(every, rateLimit{Remaining: 200, ResetAt: now.Add(30 * time.Second)}, now); d != every {
		t.Errorf("reset sooner than the interval: %v", d)
	}
	if d := refreshDelay(every, rateLimit{}, now); d != every {
		t.Errorf("no rate info yet: %v", d)
	}
}
