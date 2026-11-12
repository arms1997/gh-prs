package core

import "testing"

func TestPageSize(t *testing.T) {
	prev := []SectionData{
		{Section: Section{Title: "few"}, PRs: make([]PR, 3)},
		{Section: Section{Title: "some"}, PRs: make([]PR, 17)},
		{Section: Section{Title: "lots"}, PRs: make([]PR, 95)},
	}
	for title, want := range map[string]int{"few": 13, "some": 27, "lots": 100, "new": 100} {
		if got := pageSize(prev, title); got != want {
			t.Errorf("%s: %d, want %d", title, got, want)
		}
	}
	if got := pageSize([]SectionData{{Section: Section{Title: "empty"}}}, "empty"); got != 10 {
		t.Errorf("empty section: %d, want 10", got)
	}
}
