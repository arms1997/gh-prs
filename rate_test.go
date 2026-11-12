package main

import (
	"testing"
	"time"

	"gh-prs/core"
)

func TestRefreshDelay(t *testing.T) {
	now := time.Now()
	every := 2 * time.Minute
	reset := now.Add(40 * time.Minute)
	if d := refreshDelay(every, core.RateLimit{Remaining: 4000, ResetAt: reset}, now); d != every {
		t.Errorf("plenty left: %v", d)
	}
	if d := refreshDelay(every, core.RateLimit{Remaining: 200, ResetAt: reset}, now); d != 40*time.Minute {
		t.Errorf("low: %v, want to wait for the reset", d)
	}
	if d := refreshDelay(every, core.RateLimit{Remaining: 200, ResetAt: now.Add(30 * time.Second)}, now); d != every {
		t.Errorf("reset sooner than the interval: %v", d)
	}
	if d := refreshDelay(every, core.RateLimit{}, now); d != every {
		t.Errorf("no rate info yet: %v", d)
	}
}
