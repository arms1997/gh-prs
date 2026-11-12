package core

// What changed since you last looked: a snapshot of each PR as of the last time you selected it.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type snapshot struct {
	HeadOID        string
	CI             string
	ReviewDecision string
	Mergeable      string
	Comments       int
	Unresolved     int
}

func SnapOf(p PR) snapshot {
	return snapshot{p.HeadOID, p.CI, p.ReviewDecision, p.Mergeable, p.Comments, p.Unresolved}
}

// seenStore maps prKey → the PR as you last saw it.
type SeenStore map[string]snapshot

func seenPath() string { return filepath.Join(cacheDir(), "seen.json") }

func LoadSeen() SeenStore {
	s := SeenStore{}
	if b, err := os.ReadFile(seenPath()); err == nil {
		_ = json.Unmarshal(b, &s) // a corrupt file just means everything looks unchanged
	}
	return s
}

func (s SeenStore) Save() error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cacheDir(), 0o700); err != nil {
		return err
	}
	return os.WriteFile(seenPath(), b, 0o600)
}

// changes lists, in plain words, how a PR moved from prev to cur.
func Changes(prev, cur snapshot) []string {
	var out []string
	if cur.HeadOID != prev.HeadOID && prev.HeadOID != "" {
		out = append(out, "new commits")
	}
	if n := cur.Comments - prev.Comments; n > 0 {
		out = append(out, plural(n, "new comment", "new comments"))
	}
	if cur.CI != prev.CI && cur.CI != "" {
		out = append(out, "CI "+CIText(cur.CI))
	}
	if cur.ReviewDecision != prev.ReviewDecision && cur.ReviewDecision != "" {
		out = append(out, ReviewText(cur.ReviewDecision))
	}
	if cur.Mergeable != prev.Mergeable {
		switch {
		case cur.Mergeable == "CONFLICTING":
			out = append(out, "now has a merge conflict")
		case prev.Mergeable == "CONFLICTING":
			out = append(out, "conflict resolved")
		}
	}
	if n := cur.Unresolved - prev.Unresolved; n > 0 {
		out = append(out, plural(n, "new unresolved thread", "new unresolved threads"))
	}
	return out
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

func CIText(state string) string {
	if state == "" {
		return "no checks"
	}
	return strings.ToLower(state)
}

func ReviewText(d string) string {
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
