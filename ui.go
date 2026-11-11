package main

// The Bubble Tea model: state, messages and Update, plus the overall View.

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/cli/go-gh/v2/pkg/browser"
)

// jobTickMsg animates spinners and, every few ticks, reads new agent log lines.
type jobTickMsg struct{}

func jobTick() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg { return jobTickMsg{} })
}

type fetchedMsg struct {
	data []sectionData
	at   time.Time
	err  error
}

// refreshMsg carries the fetch generation that scheduled it, so a manual refresh doesn't start a second timer chain.
type refreshMsg int

type preparedMsg struct {
	pp  prepared
	err error
}

type doneMsg struct {
	pp  prepared
	err error
	out string
}

type confirmation struct {
	cmd Command
	pr  PR
}

type statusMsg string

type model struct {
	cfg         Config
	data        []sectionData
	rows        []Row
	collapsed   map[string]bool
	cursor      int
	offset      int
	width       int
	height      int
	detail      viewport.Model
	focusDetail bool
	loading     bool
	pendingG    bool
	status      string
	isErr       bool
	gen         int
	detailPath  string
	keySeq      string // partial command key sequence, e.g. "c" while waiting for "cc"
	confirming  *confirmation
	showHelp    bool
	jobs        map[string]*jobView // latest agent job per PR (prKey)
	spin        int
	ticking     bool
}

func prepareCmd(cfg Config, c Command, p PR, agentSession string) tea.Cmd {
	return func() tea.Msg {
		pp, err := prepare(cfg, c, p, agentSession)
		return preparedMsg{pp, err}
	}
}

func (m *model) startCommand(c Command, p PR) tea.Cmd {
	job := m.jobs[prKey(p.Repo, p.Number)]
	if job != nil && job.Status == JobRunning && (c.Background || c.Session) {
		// One agent per PR: they'd share the worktree, and a session can't be resumed while it runs.
		m.status, m.isErr = fmt.Sprintf("an agent is still running on #%d (%s)", p.Number, job.Name), true
		return nil
	}
	session := ""
	if job != nil {
		session = job.ID
	}
	m.status, m.isErr = c.Name+" #"+fmt.Sprint(p.Number)+"…", false
	return prepareCmd(m.cfg, c, p, session)
}

func (m *model) ensureTicking() tea.Cmd {
	if m.ticking {
		return nil
	}
	for _, v := range m.jobs {
		if v.Status == JobRunning {
			m.ticking = true
			return jobTick()
		}
	}
	return nil
}

func (m model) runPrepared(pp prepared) tea.Cmd {
	c := pp.execCmd()
	if pp.cmd.Interactive {
		return tea.ExecProcess(c, func(err error) tea.Msg { return doneMsg{pp: pp, err: err} })
	}
	return func() tea.Msg {
		out, err := c.CombinedOutput()
		return doneMsg{pp, err, string(out)}
	}
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(sanitize(s)), "\n")
	return lines[len(lines)-1]
}

func newModel(cfg Config) model {
	m := model{cfg: cfg, collapsed: map[string]bool{}, loading: true, detail: viewport.New(0, 0), jobs: loadJobs()}
	if data, at, ok := loadCache(cfg); ok {
		m.data = data
		m.status = "cached " + at.Format("15:04")
		m.rows = buildRows(data, cfg, m.collapsed)
	}
	return m
}

func fetchCmd(cfg Config) tea.Cmd {
	return func() tea.Msg {
		d, err := fetchAll(cfg)
		at := time.Now()
		if err == nil {
			_ = saveCache(d, at) // shortcut: a failed cache write only costs a slower next start
		}
		return fetchedMsg{d, at, err}
	}
}

func (m *model) scheduleRefresh() tea.Cmd {
	m.gen++
	if m.cfg.refresh == 0 {
		return nil
	}
	gen := m.gen
	return tea.Tick(m.cfg.refresh, func(time.Time) tea.Msg { return refreshMsg(gen) })
}

func openCmd(url string) tea.Cmd {
	return func() tea.Msg {
		if err := browser.New("", io.Discard, io.Discard).Browse(url); err != nil {
			return statusMsg("open failed: " + err.Error())
		}
		return statusMsg("opened " + url)
	}
}

func (m model) Init() tea.Cmd {
	if m.anyRunning() {
		return tea.Batch(fetchCmd(m.cfg), jobTick())
	}
	return fetchCmd(m.cfg)
}

func (m model) anyRunning() bool {
	for _, v := range m.jobs {
		if v.Status == JobRunning {
			return true
		}
	}
	return false
}

func (m model) listWidth() int  { return m.width * 66 / 100 }
func (m model) bodyHeight() int { return max(m.height-1, 1) }
func (m model) listRows() int   { return max(m.bodyHeight()-1, 1) } // minus the column header

func (m model) current() *Row {
	if m.cursor < len(m.rows) {
		return &m.rows[m.cursor]
	}
	return nil
}

func (m *model) rebuild() {
	keep := ""
	if r := m.current(); r != nil {
		keep = r.Path
	}
	m.rows = buildRows(m.data, m.cfg, m.collapsed)
	for i, r := range m.rows {
		if r.Path == keep {
			m.cursor = i
			break
		}
	}
	m.clamp()
}

// clamp keeps the cursor in range and visible, and refreshes the detail pane.
func (m *model) clamp() {
	m.cursor = max(0, min(m.cursor, len(m.rows)-1))
	h := m.listRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+h {
		m.offset = m.cursor - h + 1
	}
	m.offset = max(0, min(m.offset, len(m.rows)-h))
	m.detail.SetContent(m.renderDetail())
	if r := m.current(); r != nil && r.Path != m.detailPath {
		m.detailPath = r.Path
		m.detail.GotoTop() // only when the selection changes, so background refreshes don't reset your scroll
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.detail.Width = max(m.width-m.listWidth()-2, 1) // left border + padding
		m.detail.Height = m.bodyHeight()
		m.clamp()
	case fetchedMsg:
		m.loading = false
		if msg.err != nil {
			m.status, m.isErr = "refresh failed, showing older data: "+msg.err.Error(), true
			return m, m.scheduleRefresh()
		}
		m.data = msg.data
		m.status, m.isErr = "updated "+msg.at.Format("15:04"), false
		m.rebuild()
		return m, m.scheduleRefresh()
	case refreshMsg:
		if int(msg) == m.gen && !m.loading {
			m.loading = true
			return m, fetchCmd(m.cfg)
		}
	case statusMsg:
		m.status, m.isErr = string(msg), false
	case preparedMsg:
		if msg.err != nil {
			m.status, m.isErr = msg.pp.cmd.Name+" failed: "+msg.err.Error(), true
			return m, nil
		}
		if msg.pp.cmd.Background {
			v, err := startJob(msg.pp)
			if err != nil {
				m.status, m.isErr = msg.pp.cmd.Name+" failed to start: "+err.Error(), true
				return m, nil
			}
			m.jobs[prKey(v.Repo, v.Number)] = v
			m.status, m.isErr = fmt.Sprintf("agent started: %s #%d", v.Name, v.Number), false
			m.clamp()
			return m, m.ensureTicking()
		}
		return m, m.runPrepared(msg.pp)
	case jobTickMsg:
		m.spin++
		var cmds []tea.Cmd
		if m.spin%8 == 0 { // ~1s: read new log lines
			finished := false
			for _, v := range m.jobs {
				was := v.Status
				v.update()
				finished = finished || (was == JobRunning && v.Status != JobRunning)
			}
			m.detail.SetContent(m.renderDetail())
			if finished && !m.loading { // the agent may have pushed or edited the PR: show it now
				m.loading = true
				cmds = append(cmds, fetchCmd(m.cfg))
			}
		}
		if m.anyRunning() {
			cmds = append(cmds, jobTick())
		} else {
			m.ticking = false
		}
		return m, tea.Batch(cmds...)
	case doneMsg:
		if msg.pp.cmd.Prompt != "" && !msg.pp.cmd.Terminal { // terminal launchers delete their own prompt file
			for _, e := range msg.pp.env {
				if f, ok := strings.CutPrefix(e, "PR_PROMPT_FILE="); ok {
					os.Remove(f)
				}
			}
		}
		label := fmt.Sprintf("%s #%d", msg.pp.cmd.Name, msg.pp.pr.Number)
		if msg.err != nil {
			detail := lastLine(msg.out)
			if detail == "" {
				detail = msg.err.Error()
			}
			m.status, m.isErr = "✗ "+label+": "+detail, true
			return m, nil
		}
		m.status, m.isErr = "✓ "+label, false
		if msg.pp.cmd.Refresh && !m.loading {
			m.loading = true
			return m, fetchCmd(m.cfg)
		}
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return ""
	}
	h, lw := m.bodyHeight(), m.listWidth()
	lines := []string{columnHeader(lw)}
	for i := m.offset; i < len(m.rows) && i < m.offset+m.listRows(); i++ {
		lines = append(lines, m.renderRow(m.rows[i], i == m.cursor, m.shaded(i), lw))
	}
	if len(m.rows) == 0 && !m.loading {
		lines = append(lines, dim.Render("No PRs match your sections."))
	}
	list := lipgloss.NewStyle().Width(lw).Height(h).MaxHeight(h).Render(strings.Join(lines, "\n"))

	borderColor := lipgloss.Color("8")
	if m.focusDetail {
		borderColor = lipgloss.Color("5")
	}
	right := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).BorderLeft(true).BorderForeground(borderColor).
		PaddingLeft(1).Height(h).MaxHeight(h).
		Render(m.detail.View())

	return lipgloss.JoinHorizontal(lipgloss.Top, list, right) + "\n" + m.footer()
}

func (m model) helpText() string {
	var b strings.Builder
	b.WriteString(bold.Render("Keys") + "  " + dim.Render("(? to close)") + "\n\n")
	for _, kv := range [][2]string{
		{"j k ↑ ↓", "move"}, {"gg G", "top / bottom"}, {"ctrl+d ctrl+u", "half page"},
		{"enter h l", "fold / collapse / expand"}, {"e", "focus details (esc back)"},
		{"o", "open in browser"}, {"r", "refresh"}, {"q", "quit"},
	} {
		fmt.Fprintf(&b, "%-14s %s\n", kv[0], kv[1])
	}
	b.WriteString("\n" + bold.Render("Commands") + "  " + dim.Render(configPath()) + "\n\n")
	for _, c := range m.cfg.Commands {
		fmt.Fprintf(&b, "%-14s %s\n", c.Key, c.Name)
	}
	return b.String()
}

func (m model) footer() string {
	if c := m.confirming; c != nil {
		return yellow.Bold(true).Render(fmt.Sprintf("%s #%d? y/N", c.cmd.Name, c.pr.Number))
	}
	help := "j/k move · enter fold · e details · o browser · cc agent · ct take over · ? keys · q quit"
	overflow := m.detail.TotalLineCount() > m.detail.Height
	pct := fmt.Sprintf("%d%%", int(m.detail.ScrollPercent()*100))
	if m.focusDetail {
		help = "j/k scroll · o browser · esc back"
		if overflow {
			help = "j/k scroll (" + pct + ") · o browser · esc back"
		}
	} else if overflow {
		help = "e scroll details (" + pct + ") · " + help
	}
	if m.keySeq != "" {
		help = m.keySeq + "…"
		for _, c := range m.cfg.Commands {
			if strings.HasPrefix(c.Key, m.keySeq) {
				help += "  " + c.Key + " " + c.Name
			}
		}
	}
	status := m.status
	if m.loading {
		status = strings.TrimPrefix(status+" · ", " · ") + "refreshing…"
	}
	st := dim
	if m.isErr {
		st = red
	}
	line := help
	if status != "" {
		line += "  │  " + status
	}
	return st.Render(ansi.Truncate(line, m.width, "…"))
}
