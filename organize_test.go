package main

import (
	"os"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"gh-prs/core"
)

func orgModel(t *testing.T) model {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := filterModel(t) // PRs o/web#1 "fix login" and o/api#2 "add cache"
	m.org = &core.Organize{}
	m.data[0].Section.GroupBy = []string{"project"}
	m.rebuild()
	return m
}

func selectPR(m model, n int) model {
	for i, r := range m.rows {
		if r.PR != nil && r.PR.Number == n {
			m.cursor = i
		}
	}
	m.clamp()
	return m
}

func TestPinHideSnooze(t *testing.T) {
	m := orgModel(t)
	if got := prNumbers(m.rows); !reflect.DeepEqual(got, []int{1, 2}) {
		t.Fatalf("start: %v", got)
	}

	m = typeKeys(selectPR(m, 2), runes("p"), runes("p"))
	if got := prNumbers(m.rows); !reflect.DeepEqual(got, []int{2, 1}) {
		t.Fatalf("pinned #2 should come first: %v", got)
	}
	if o, _ := core.LoadOrganize(); !o.IsPinned("o/api#2") {
		t.Fatal("pin should be saved to organize.yml")
	}

	m = typeKeys(selectPR(m, 1), runes("p"), runes("h"))
	if got := prNumbers(m.rows); !reflect.DeepEqual(got, []int{2}) {
		t.Fatalf("hidden #1 should be gone: %v", got)
	}
	m = typeKeys(m, runes("p"), runes("a"))
	if got := prNumbers(m.rows); len(got) != 2 {
		t.Fatalf("pa should show hidden PRs: %v", got)
	}
	m = typeKeys(selectPR(m, 1), runes("p"), runes("h"), runes("p"), runes("a"))
	if got := prNumbers(m.rows); len(got) != 2 {
		t.Fatalf("ph again should unhide: %v", got)
	}

	m = typeKeys(selectPR(m, 1), runes("p"), runes("s"))
	if got := prNumbers(m.rows); !reflect.DeepEqual(got, []int{2}) {
		t.Fatalf("snoozed #1 should be gone: %v", got)
	}
	data := []core.SectionData{{Section: m.data[0].Section, PRs: append([]core.PR(nil), m.data[0].PRs...)}}
	data[0].PRs[0].Comments++ // something happened on #1
	tm, _ := m.Update(fetchedMsg{data: data})
	m = tm.(model)
	if got := prNumbers(m.rows); len(got) != 2 || m.org.IsSnoozed("o/web#1") {
		t.Fatalf("a change should wake #1: rows %v, still snoozed %v", got, m.org.IsSnoozed("o/web#1"))
	}
}

func TestMoveToProject(t *testing.T) {
	m := typeKeys(selectPR(orgModel(t), 1), runes("p"), runes("m"))
	if m.input == nil {
		t.Fatal("pm should prompt for a project")
	}
	m = typeKeys(m, runes("cc"), tea.KeyMsg{Type: tea.KeyEnter}) // "cc" goes into the prompt, not to commands
	if got := titles(m.rows); strings.Join(got, ",") != "Mine,cc,#1,No project,#2" {
		t.Fatalf("rows: %v", got)
	}
	if m.keySeq != "" || m.org.Projects["o/web#1"] != "cc" {
		t.Fatalf("keySeq=%q project=%q", m.keySeq, m.org.Projects["o/web#1"])
	}
}

const answerOK = `Moved both PRs into one project and added a section.
<config.yml>
sections:
  - title: Mine
    query: is:pr author:@me
    group_by: [project, turn]
</config.yml>
<organize.yml>
projects:
  o/web#1: login
pinned: [o/api#2]
snoozed:
  o/web#9: {}
</organize.yml>`

func TestApplyProposal(t *testing.T) {
	m := orgModel(t)
	os.MkdirAll(strings.TrimSuffix(core.ConfigPath(), "/config.yml"), 0o755)
	os.WriteFile(core.ConfigPath(), []byte("refresh_interval: 2m\n"), 0o644)
	oldOrg, _ := m.org.Marshal()
	p, err := core.BuildProposal("x", answerOK, []byte("refresh_interval: 2m\n"), oldOrg, m.org)
	if err != nil {
		t.Fatal(err)
	}
	m.proposal = p
	tm, _ := m.handleKey(runes("y"))
	m = tm.(model)
	if m.proposal != nil || m.cfg.Sections[0].GroupBy[1] != "turn" || m.org.Projects["o/web#1"] != "login" {
		t.Fatalf("not applied: cfg=%+v org=%+v status=%q", m.cfg.Sections, m.org, m.status)
	}
	if b, _ := os.ReadFile(core.ConfigPath() + ".bak"); string(b) != "refresh_interval: 2m\n" {
		t.Fatalf("backup: %q", b)
	}
	if o, _ := core.LoadOrganize(); o.Projects["o/web#1"] != "login" {
		t.Fatal("organize.yml not saved")
	}
}

func titles(rows []core.Row) []string {
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

// A key that ends no command (like the old "cd") says so instead of doing nothing.
func TestDeadKeySequence(t *testing.T) {
	m := selectPR(orgModel(t), 1)
	m.cfg.Commands = []core.Command{{Key: "cc", Name: "agent", Run: "true"}}
	for _, seq := range []string{"cd", "pz"} {
		m := typeKeys(m, runes(seq[:1]), runes(seq[1:]))
		if m.keySeq != "" || !m.isErr || !strings.Contains(m.status, "no command "+seq) {
			t.Fatalf("%s: keySeq=%q status=%q", seq, m.keySeq, m.status)
		}
	}
}
