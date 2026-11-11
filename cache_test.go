package main

import (
	"os"
	"testing"
	"time"
)

func TestCacheRoundTripAndInvalidation(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	cfg := Config{Sections: []Section{{Title: "Mine", Query: "is:pr author:@me"}}}
	at := time.Date(2026, 11, 11, 9, 30, 0, 0, time.UTC)

	if err := saveCache([]sectionData{{Section: cfg.Sections[0], PRs: []PR{{Number: 7, Title: "x"}}}}, at); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(cachePath()); info.Mode().Perm() != 0o600 {
		t.Fatalf("cache perms = %v, want 0600", info.Mode().Perm())
	}

	data, gotAt, ok := loadCache(cfg)
	if !ok || !gotAt.Equal(at) || len(data) != 1 || data[0].PRs[0].Number != 7 {
		t.Fatalf("round trip: ok=%v at=%v data=%+v", ok, gotAt, data)
	}

	cfg.Sections[0].Query = "is:pr author:@me label:x"
	if _, _, ok := loadCache(cfg); ok {
		t.Fatal("expected a changed query to miss the cache")
	}
}

func TestRefreshInterval(t *testing.T) {
	base := "sections:\n  - title: x\n    query: is:pr\n"
	cases := map[string]time.Duration{"": 2 * time.Minute, "refresh_interval: 5m\n": 5 * time.Minute, "refresh_interval: \"0\"\n": 0}
	for extra, want := range cases {
		cfg, err := parseConfig([]byte(extra + base))
		if err != nil || cfg.refresh != want {
			t.Errorf("%q: got %v, %v; want %v", extra, cfg.refresh, err, want)
		}
	}
	for _, bad := range []string{"refresh_interval: 5s\n", "refresh_interval: soon\n"} {
		if _, err := parseConfig([]byte(bad + base)); err == nil {
			t.Errorf("%q: expected error", bad)
		}
	}
}
