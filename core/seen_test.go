package core

import (
	"reflect"
	"testing"
)

func TestBaselineThenChanges(t *testing.T) {
	p := PR{Repo: "o/r", Number: 1, HeadOID: "a", CI: "PENDING"}
	s := SeenStore{}
	if !s.Baseline([]SectionData{{PRs: []PR{p}}}) {
		t.Fatal("a new PR wasn't recorded")
	}
	p.HeadOID, p.CI, p.Comments = "b", "SUCCESS", 2
	if s.Baseline([]SectionData{{PRs: []PR{p}}}) {
		t.Fatal("a PR already seen was recorded again")
	}
	got := Changes(s[PrKey("o/r", 1)], SnapOf(p))
	if want := []string{"new commits", "2 new comments", "CI success"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("changes %v, want %v", got, want)
	}
}
