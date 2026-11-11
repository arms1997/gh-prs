package main

import (
	"reflect"
	"testing"
)

func TestStatusOf(t *testing.T) {
	cases := []struct {
		name string
		pr   PR
		want Status
	}{
		{"draft wins over everything", PR{IsDraft: true, CI: "FAILURE", ReviewDecision: "APPROVED"}, Draft},
		{"changes requested", PR{ReviewDecision: "CHANGES_REQUESTED", CI: "SUCCESS"}, NeedsAction},
		{"ci failing", PR{ReviewDecision: "APPROVED", CI: "FAILURE"}, NeedsAction},
		{"ci error", PR{CI: "ERROR"}, NeedsAction},
		{"conflict", PR{ReviewDecision: "APPROVED", CI: "SUCCESS", Mergeable: "CONFLICTING"}, NeedsAction},
		{"approved and green", PR{ReviewDecision: "APPROVED", CI: "SUCCESS"}, ReadyToMerge},
		{"approved, no checks", PR{ReviewDecision: "APPROVED"}, ReadyToMerge},
		{"approved, ci pending", PR{ReviewDecision: "APPROVED", CI: "PENDING"}, WaitingReview},
		{"review required", PR{ReviewDecision: "REVIEW_REQUIRED", CI: "SUCCESS"}, WaitingReview},
	}
	for _, c := range cases {
		if got := statusOf(c.pr); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func titles(rows []Row) []string {
	var out []string
	for _, r := range rows {
		if r.PR != nil {
			out = append(out, "#"+string(rune('0'+r.PR.Number)))
		} else {
			out = append(out, r.Title)
		}
	}
	return out
}

func TestBuildRowsNestsAndOrders(t *testing.T) {
	cfg := Config{ProjectLabelPrefix: "proj:"}
	data := []sectionData{{
		Section: Section{Title: "Mine", GroupBy: []string{"project", "status"}},
		PRs: []PR{
			{Number: 1, Labels: []string{"bug"}},
			{Number: 2, Labels: []string{"proj:beta"}, IsDraft: true},
			{Number: 3, Labels: []string{"proj:alpha"}, ReviewDecision: "APPROVED", CI: "SUCCESS"},
			{Number: 4, Labels: []string{"proj:beta"}, CI: "FAILURE"},
		},
	}}

	got := titles(buildRows(data, cfg, map[string]bool{}, ""))
	want := []string{
		"Mine",
		"alpha", "Ready to merge", "#3",
		"beta", "Needs action", "#4", "Draft", "#2",
		"No project", "Waiting for review", "#1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rows:\n got %v\nwant %v", got, want)
	}

	got = titles(buildRows(data, cfg, map[string]bool{"Mine/beta": true}, ""))
	want = []string{"Mine", "alpha", "Ready to merge", "#3", "beta", "No project", "Waiting for review", "#1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("collapsed rows:\n got %v\nwant %v", got, want)
	}
}

func TestStacking(t *testing.T) {
	cfg := Config{}
	pr := func(n int, head, base string, draft bool) PR {
		return PR{Number: n, Repo: "o/r", HeadRefName: head, BaseRefName: base, IsDraft: draft}
	}
	data := []sectionData{{
		Section: Section{Title: "Mine", GroupBy: []string{"status"}},
		PRs: []PR{
			pr(3, "c", "b", false), // stacked on #2, listed before it
			pr(1, "a", "main", false),
			pr(2, "b", "a", false),
			pr(4, "d", "a", true),                          // stacked on #1, but lands in Draft
			pr(5, "x", "y", false), pr(6, "y", "x", false), // cycle must not loop or drop PRs
		},
	}}

	type got struct{ num, stack, on int }
	var rows []got
	for _, r := range buildRows(data, cfg, map[string]bool{}, "") {
		if r.PR != nil {
			rows = append(rows, got{r.PR.Number, r.Stack, r.StackedOn})
		}
	}
	want := []got{
		{1, 0, 0}, {2, 1, 1}, {3, 2, 2}, // Waiting for review: 1 ↳2 ↳3
		{5, 0, 6}, {6, 1, 5},
		{4, 0, 1}, // Draft: parent #1 lives in another group
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("got %v\nwant %v", rows, want)
	}
}

func TestSanitizeStripsEscapes(t *testing.T) {
	if got := sanitize("ok\x1b]0;pwned\x07\r\nnext\u009b"); got != "ok]0;pwned\nnext" {
		t.Fatalf("got %q", got)
	}
}

func TestDefaultConfigParses(t *testing.T) {
	if _, err := parseConfig(mustRead("defaults/config.yml")); err != nil {
		t.Fatal(err)
	}
	if _, err := parseConfig([]byte("sections:\n  - title: x\n    query: is:pr\n    group_by: [nope]\n")); err == nil {
		t.Fatal("expected unknown group_by to fail")
	}
}
