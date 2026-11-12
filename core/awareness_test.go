package core

import (
	"reflect"
	"strings"
	"testing"
)

func TestTurnOf(t *testing.T) {
	cases := []struct {
		name string
		Pr   PR
		mine bool
		why  string
	}{
		{"my draft", PR{Mine: true, IsDraft: true}, true, "it's a draft"},
		{"my failing PR", PR{Mine: true, CI: "FAILURE"}, true, "CI failing"},
		{"my PR, someone replied", PR{Mine: true, CI: "SUCCESS", UnresolvedNotMine: 2}, true, "2 threads waiting on your reply"},
		{"my PR, ready", PR{Mine: true, ReviewDecision: "APPROVED", CI: "SUCCESS"}, true, "ready to merge"},
		{"my PR, waiting", PR{Mine: true, ReviewDecision: "REVIEW_REQUIRED", CI: "SUCCESS"}, false, "waiting for review"},
		{"review request", PR{}, true, "your review is requested"},
		{"already reviewed", PR{ReviewedSincePush: true}, false, "you reviewed it; waiting for new commits"},
	}
	for _, c := range cases {
		if mine, why := TurnOf(c.Pr); mine != c.mine || why != c.why {
			t.Errorf("%s: got %v %q, want %v %q", c.name, mine, why, c.mine, c.why)
		}
	}
}

func TestChanges(t *testing.T) {
	prev := snapshot{HeadOID: "a", CI: "SUCCESS", ReviewDecision: "REVIEW_REQUIRED", Mergeable: "MERGEABLE", Comments: 2}
	cur := snapshot{HeadOID: "b", CI: "FAILURE", ReviewDecision: "CHANGES_REQUESTED", Mergeable: "CONFLICTING", Comments: 5, Unresolved: 1}
	want := []string{"new commits", "3 new comments", "CI failure", "changes requested", "now has a merge conflict", "1 new unresolved thread"}
	if got := Changes(prev, cur); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q\nwant %q", got, want)
	}
	if got := Changes(cur, cur); len(got) != 0 {
		t.Fatalf("no change should list nothing: %q", got)
	}
}

func TestLinearID(t *testing.T) {
	for _, c := range []struct{ title, branch, want string }{
		{"docs(cli): voice tool mapping [AGT-915]", "x", "AGT-915"},
		{"fix: thing", "armeenhadizadeh/agt-1490-tool-mapping", "AGT-1490"},
		{"fix: thing", "main-fix", ""},
	} {
		if got := linearID(c.title, c.branch); got != c.want {
			t.Errorf("%q %q: got %q, want %q", c.title, c.branch, got, c.want)
		}
	}
	if LinearURL("go-augment", "AGT-915") != "https://linear.app/go-augment/issue/AGT-915" || LinearURL("", "AGT-1") != "" {
		t.Error("linearURL")
	}
}

func TestFailureExcerpt(t *testing.T) {
	line := func(s string) string { return "build\tUNKNOWN STEP\t2026-11-11T20:01:02.1234567Z " + s + "\n" }
	var raw strings.Builder
	for i := 0; i < 100; i++ {
		raw.WriteString(line("##[debug]noise"))
		raw.WriteString(line("setup step output"))
	}
	raw.WriteString(line("> tsc --noEmit"))
	raw.WriteString(line("##[error]src/a.ts(3,1): error TS2322: \x1b[31mbad type\x1b[0m"))
	raw.WriteString(line("##[error]Process completed with exit code 2."))
	for i := 0; i < 50; i++ {
		raw.WriteString(line("Post job cleanup."))
	}
	got := failureExcerpt(raw.String(), 10)
	if !strings.Contains(got, "> tsc --noEmit\n✗ src/a.ts(3,1): error TS2322: bad type\n✗ Process completed") || strings.Contains(got, "debug") {
		t.Fatalf("should center on the errors:\n%s", got)
	}
	if strings.Contains(got, "Post job cleanup") || !strings.HasSuffix(got, "✗ Process completed with exit code 2.") {
		t.Fatalf("should end at the last error:\n%s", got)
	}
	if got := failureExcerpt("a\nb\nc\n", 2); got != "b\nc" {
		t.Fatalf("no errors marked: should be the tail, got %q", got)
	}
}

func TestTurnGroupBy(t *testing.T) {
	cfg, err := ParseConfig([]byte("sections:\n  - title: x\n    query: is:pr\n    group_by: [turn]\n"))
	if err != nil {
		t.Fatal(err)
	}
	data := []SectionData{{Section: cfg.Sections[0], PRs: []PR{{Number: 1, Mine: true, CI: "SUCCESS", ReviewDecision: "REVIEW_REQUIRED"}, {Number: 2, Mine: true, IsDraft: true}}}}
	if got := titles(BuildRows(data, cfg, View{})); strings.Join(got, ",") != "x,On me,#2,On them,#1" {
		t.Fatalf("got %v", got)
	}
}

func TestCheckState(t *testing.T) {
	for in, want := range map[string]string{"SUCCESS": "pass", "FAILURE": "fail", "TIMED_OUT": "fail", "": "pending", "PENDING": "pending", "SKIPPED": "skipped", "NEUTRAL": "skipped"} {
		if got := checkState(in); got != want {
			t.Errorf("checkState(%q) = %q, want %q", in, got, want)
		}
	}
}
