package main

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"gh-prs/core"
)

func filterModel(t *testing.T) model {
	t.Helper()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	cfg, err := core.ParseConfig([]byte("sections:\n  - title: Mine\n    query: is:pr\n"))
	if err != nil {
		t.Fatal(err)
	}
	m := model{cfg: cfg, collapsed: map[string]bool{}, seen: core.SeenStore{}, logs: map[string]*failLog{}, jobs: map[string]*core.JobView{}}
	m.data = []core.SectionData{{Section: cfg.Sections[0], PRs: []core.PR{
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

func prNumbers(rows []core.Row) []int {
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
	if reloaded := core.LoadSeen(); reloaded[core.PrKey("o/api", 2)] != core.SnapOf(pr2) {
		t.Fatal("seen state should persist")
	}
}
