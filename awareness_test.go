package main

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTurnOf(t *testing.T) {
	cases := []struct {
		name string
		pr   PR
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
		if mine, why := turnOf(c.pr); mine != c.mine || why != c.why {
			t.Errorf("%s: got %v %q, want %v %q", c.name, mine, why, c.mine, c.why)
		}
	}
}

func TestChanges(t *testing.T) {
	prev := snapshot{HeadOID: "a", CI: "SUCCESS", ReviewDecision: "REVIEW_REQUIRED", Mergeable: "MERGEABLE", Comments: 2}
	cur := snapshot{HeadOID: "b", CI: "FAILURE", ReviewDecision: "CHANGES_REQUESTED", Mergeable: "CONFLICTING", Comments: 5, Unresolved: 1}
	want := []string{"new commits", "3 new comments", "CI failure", "changes requested", "now has a merge conflict", "1 new unresolved thread"}
	if got := changes(prev, cur); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q\nwant %q", got, want)
	}
	if got := changes(cur, cur); len(got) != 0 {
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
	if linearURL("go-augment", "AGT-915") != "https://linear.app/go-augment/issue/AGT-915" || linearURL("", "AGT-1") != "" {
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

func filterModel(t *testing.T) model {
	t.Helper()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	cfg, err := parseConfig([]byte("sections:\n  - title: Mine\n    query: is:pr\n"))
	if err != nil {
		t.Fatal(err)
	}
	m := model{cfg: cfg, collapsed: map[string]bool{}, seen: seenStore{}, logs: map[string]*failLog{}, jobs: map[string]*jobView{}}
	m.data = []sectionData{{Section: cfg.Sections[0], PRs: []PR{
		{Number: 1, Repo: "o/web", Title: "fix login", HeadRefName: "a", Linear: "AGT-1"},
		{Number: 2, Repo: "o/api", Title: "add cache", HeadRefName: "b"},
	}}}
	m.rebuild()
	return m
}

func typeKeys(m model, keys ...tea.KeyMsg) model {
	for _, k := range keys {
		tm, _ := m.handleKey(k)
		m = tm.(model)
	}
	return m
}

func runes(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func prNumbers(rows []Row) []int {
	var out []int
	for _, r := range rows {
		if r.PR != nil {
			out = append(out, r.PR.Number)
		}
	}
	return out
}

// Typing into the filter narrows the list, and keys like "cc" go into the filter, not to commands.
func TestFilter(t *testing.T) {
	m := typeKeys(filterModel(t), runes("/"), runes("c"), runes("c"))
	if !m.filtering || m.filter != "cc" || m.keySeq != "" || m.status != "" {
		t.Fatalf("filtering=%v filter=%q keySeq=%q status=%q", m.filtering, m.filter, m.keySeq, m.status)
	}

	m = typeKeys(filterModel(t), runes("/"), runes("agt"), tea.KeyMsg{Type: tea.KeyEnter})
	if got := prNumbers(m.rows); !reflect.DeepEqual(got, []int{1}) || m.filtering {
		t.Fatalf("filter by Linear ID: rows %v, filtering %v", got, m.filtering)
	}
	m = typeKeys(m, tea.KeyMsg{Type: tea.KeyEsc})
	if got := prNumbers(m.rows); len(got) != 2 || m.filter != "" {
		t.Fatalf("esc should clear the filter: %v %q", got, m.filter)
	}

	m = typeKeys(filterModel(t), runes("/"), runes("o/api"), tea.KeyMsg{Type: tea.KeySpace}, runes("cache"))
	if got := prNumbers(m.rows); !reflect.DeepEqual(got, []int{2}) {
		t.Fatalf("two terms: %v (filter %q)", got, m.filter)
	}
}

// A PR is new when it changed since you last selected it; selecting it shows what changed and clears the mark.
func TestSeen(t *testing.T) {
	m := filterModel(t)
	m.baseline()
	m.cursor = 0 // the section header: nothing selected yet
	m.clamp()
	m.data[0].PRs[1].CI = "FAILURE"
	m.data[0].PRs[1].Comments = 1
	m.rebuild()
	pr2 := m.data[0].PRs[1]
	if !m.isNew(pr2) || m.isNew(m.data[0].PRs[0]) {
		t.Fatal("only the changed PR should be new")
	}
	for i, r := range m.rows {
		if r.PR != nil && r.PR.Number == 2 {
			m.cursor = i
		}
	}
	m.clamp()
	if m.isNew(pr2) || strings.Join(m.recent, " · ") != "1 new comment · CI failure" {
		t.Fatalf("after selecting: new=%v recent=%q", m.isNew(pr2), m.recent)
	}
	if reloaded := loadSeen(); reloaded[prKey("o/api", 2)] != snapOf(pr2) {
		t.Fatal("seen state should persist")
	}
}

func TestTurnGroupBy(t *testing.T) {
	cfg, err := parseConfig([]byte("sections:\n  - title: x\n    query: is:pr\n    group_by: [turn]\n"))
	if err != nil {
		t.Fatal(err)
	}
	data := []sectionData{{Section: cfg.Sections[0], PRs: []PR{{Number: 1, Mine: true, CI: "SUCCESS", ReviewDecision: "REVIEW_REQUIRED"}, {Number: 2, Mine: true, IsDraft: true}}}}
	if got := titles(buildRows(data, cfg, map[string]bool{}, "")); strings.Join(got, ",") != "x,On me,#2,On them,#1" {
		t.Fatalf("got %v", got)
	}
}
