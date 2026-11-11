package main

// Keyboard handling, including command key sequences and y/N confirmation.

import (
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
		return m, fetchCmd(m.cfg)
	default:
		return m, nil
	}
	m.clamp()
	return m, nil
}
