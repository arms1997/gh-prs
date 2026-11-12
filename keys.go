package main

// Keyboard handling, including command key sequences and y/N confirmation.

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// commandFor matches a key sequence: the command if complete, or partial=true if it's a prefix of one.
func (m model) commandFor(seq string) (cmd *Command, partial bool) {
	for i, c := range m.cfg.Commands {
		if c.Key == seq {
			return &m.cfg.Commands[i], false
		}
		if strings.HasPrefix(c.Key, seq) {
			partial = true
		}
	}
	return nil, partial
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	if k == "ctrl+c" {
		return m, tea.Quit
	}
	r := m.current()

	if in := m.input; in != nil { // typing goes into the prompt, never into commands
		switch msg.Type {
		case tea.KeyEsc:
			m.input = nil
		case tea.KeyEnter:
			m.input = nil
			return m, in.submit(&m, strings.TrimSpace(in.value))
		case tea.KeyBackspace:
			if rs := []rune(in.value); len(rs) > 0 {
				in.value = string(rs[:len(rs)-1])
			}
		case tea.KeyRunes:
			in.value += string(msg.Runes)
		case tea.KeySpace:
			in.value += " "
		}
		return m, nil
	}

	if p := m.proposal; p != nil {
		switch {
		case p.loading:
			if k == "esc" {
				m.proposal = nil
				m.clamp()
			}
			return m, nil
		case k == "y":
			return m, m.applyProposal()
		case k == "n" || k == "esc":
			m.proposal = nil
			m.status, m.isErr = "discarded Claude's proposal", false
			m.clamp()
			return m, nil
		}
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd
	}

	if m.orgKey {
		m.orgKey = false
		return m.organizeKey(k, r)
	}

	if m.filtering { // typing goes into the filter, never into commands
		switch msg.Type {
		case tea.KeyEsc:
			m.filter, m.filtering = "", false
		case tea.KeyEnter:
			m.filtering = false
		case tea.KeyBackspace:
			if rs := []rune(m.filter); len(rs) > 0 {
				m.filter = string(rs[:len(rs)-1])
			}
		case tea.KeyCtrlU:
			m.filter = ""
		case tea.KeyRunes, tea.KeySpace:
			m.filter += string(msg.Runes)
			if msg.Type == tea.KeySpace {
				m.filter += " "
			}
		default:
			return m, nil
		}
		m.rebuild()
		return m, nil
	}

	if c := m.confirming; c != nil {
		m.confirming = nil
		if k == "y" || k == "Y" {
			return m, m.startCommand(c.cmd, c.pr)
		}
		m.status, m.isErr = "cancelled", false
		return m, nil
	}

	if seq := m.keySeq + k; m.keySeq != "" || !builtinKeys[k] {
		cmd, partial := m.commandFor(seq)
		switch {
		case cmd != nil:
			m.keySeq = ""
			if r == nil || r.PR == nil {
				m.status, m.isErr = "select a PR first", false
				return m, nil
			}
			if cmd.Confirm {
				m.confirming = &confirmation{*cmd, *r.PR}
				return m, nil
			}
			return m, m.startCommand(*cmd, *r.PR)
		case partial:
			m.keySeq = seq
			return m, nil
		case m.keySeq != "":
			m.keySeq = "" // dead sequence: drop it and treat this key on its own
			return m.handleKey(msg)
		}
	}

	if k == "?" {
		m.showHelp = !m.showHelp
		m.detail.SetContent(m.renderDetail())
		m.detail.GotoTop()
		return m, nil
	}

	if m.focusDetail {
		switch k {
		case "esc", "e", "q", "h", "left":
			m.focusDetail = false
			return m, nil
		case "o":
			if r != nil && r.PR != nil {
				return m, openCmd(r.PR.URL)
			}
		}
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd
	}

	if k == "g" {
		if m.pendingG {
			m.pendingG = false
			m.cursor = 0
			m.clamp()
		} else {
			m.pendingG = true
		}
		return m, nil
	}
	m.pendingG = false

	switch k {
	case "p":
		m.orgKey = true
		return m, nil
	case "/":
		m.filtering = true
		return m, nil
	case "esc":
		if m.filter != "" {
			m.filter = ""
			m.rebuild()
		}
		return m, nil
	case "f":
		if r != nil && r.PR != nil {
			return m, m.loadFailLog(*r.PR)
		}
		return m, nil
	case "i":
		if r != nil && r.PR != nil {
			if url := linearURL(m.cfg.LinearWorkspace, r.PR.Linear); url != "" {
				return m, openCmd(url)
			}
			m.status, m.isErr = "no Linear issue found in the title or branch", false
		}
		return m, nil
	case "q":
		return m, tea.Quit
	case "j", "down":
		m.cursor++
	case "k", "up":
		m.cursor--
	case "G":
		m.cursor = len(m.rows) - 1
	case "ctrl+d":
		m.cursor += m.listRows() / 2
	case "ctrl+u":
		m.cursor -= m.listRows() / 2
	case "enter", " ", "tab":
		if r != nil && r.PR == nil {
			m.collapsed[r.Path] = !m.collapsed[r.Path]
			m.rebuild()
		}
		return m, nil
	case "h", "left":
		if r != nil && r.PR == nil {
			m.collapsed[r.Path] = true
			m.rebuild()
		}
		return m, nil
	case "l", "right":
		if r != nil && r.PR == nil {
			m.collapsed[r.Path] = false
			m.rebuild()
		} else if r != nil {
			m.focusDetail = true
		}
		return m, nil
	case "e":
		if r != nil && r.PR != nil {
			m.focusDetail = true
		}
		return m, nil
	case "o":
		if r != nil && r.PR != nil {
			return m, openCmd(r.PR.URL)
		}
		return m, nil
	case "r":
		if m.loading {
			return m, nil
		}
		m.loading = true
		return m, fetchCmd(m.cfg, m.data)
	default:
		return m, nil
	}
	m.clamp()
	return m, nil
}

// organizeKey runs the action picked after p.
func (m model) organizeKey(k string, r *Row) (tea.Model, tea.Cmd) {
	switch k {
	case "a":
		m.showAll = !m.showAll
		m.rebuild()
		return m, nil
	case "c":
		m.input = &textInput{label: "Reorganize: ", submit: func(m *model, ask string) tea.Cmd { return m.askClaude(ask) }}
		return m, nil
	}
	if r == nil || r.PR == nil {
		if strings.Contains("psmh", k) {
			m.status, m.isErr = "select a PR first", false
		}
		return m, nil
	}
	p := *r.PR
	key := prKey(p.Repo, p.Number)
	var note string
	switch k {
	case "p":
		note = map[bool]string{true: "pinned", false: "unpinned"}[toggle(&m.org.Pinned, key)]
	case "h":
		note = map[bool]string{true: "hidden (pa shows hidden PRs)", false: "unhidden"}[toggle(&m.org.Hidden, key)]
	case "s":
		if m.org.snoozed(key) {
			delete(m.org.Snoozed, key)
			note = "unsnoozed"
		} else {
			m.org.snooze(p)
			note = "snoozed until something changes on it"
		}
	case "m":
		m.input = &textInput{label: "Move to project: ", value: m.org.Projects[key], submit: func(m *model, name string) tea.Cmd {
			m.org.setProject(key, name)
			m.saveOrganize(fmt.Sprintf("#%d moved to %s", p.Number, map[bool]string{true: name, false: "its label's project"}[name != ""]))
			return nil
		}}
		return m, nil
	default:
		return m, nil
	}
	m.saveOrganize(fmt.Sprintf("#%d %s", p.Number, note))
	return m, nil
}

func (m *model) saveOrganize(note string) {
	if err := m.org.save(); err != nil {
		m.status, m.isErr = "couldn't save "+organizePath()+": "+err.Error(), true
	} else {
		m.status, m.isErr = note, false
	}
	m.rebuild()
}
