package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type cacheFile struct {
	FetchedAt time.Time
	Sections  []cachedSection
}

type cachedSection struct {
	Title string
	Query string
	PRs   []PR
}

func cacheDir() string {
	dir := os.Getenv("XDG_CACHE_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".cache")
	}
	return filepath.Join(dir, "gh-prs")
}

func cachePath() string { return filepath.Join(cacheDir(), "prs.json") }

// loadCache returns cached data only if every configured section is present, so a config edit forces a fresh view.
func loadCache(cfg Config) ([]sectionData, time.Time, bool) {
	raw, err := os.ReadFile(cachePath())
	if err != nil {
		return nil, time.Time{}, false
	}
	var c cacheFile
	if json.Unmarshal(raw, &c) != nil {
		return nil, time.Time{}, false
	}
	var out []sectionData
	for _, s := range cfg.Sections {
		found := false
		for _, cs := range c.Sections {
			if cs.Title == s.Title && cs.Query == s.Query {
				out = append(out, sectionData{Section: s, PRs: cs.PRs})
				found = true
				break
			}
		}
		if !found {
			return nil, time.Time{}, false
		}
	}
	return out, c.FetchedAt, true
}

// saveCache writes atomically with owner-only permissions, since it holds private PR descriptions.
func saveCache(data []sectionData, at time.Time) error {
	c := cacheFile{FetchedAt: at}
	for _, sd := range data {
		c.Sections = append(c.Sections, cachedSection{Title: sd.Section.Title, Query: sd.Section.Query, PRs: sd.PRs})
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return err
	}
	path := cachePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "prs-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
