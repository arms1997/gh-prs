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

	"gh-prs/core"
)

// jobTickMsg animates spinners and, every few ticks, reads new agent log lines.
type jobTickMsg struct{}

func jobTick() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg { return jobTickMsg{} })
}

type fetchedMsg struct {
	data []core.SectionData
	at   time.Time
	rate core.RateLimit
	err  error
}

// refreshMsg carries the fetch generation that scheduled it, so a manual refresh doesn't start a second timer chain.
type refreshMsg int

type preparedMsg struct {
	pp  core.Prepared
	err error
}

type doneMsg struct {
	pp  core.Prepared
	err error
	out string
}

type confirmation struct {
	cmd core.Command
	pr  core.PR
}

type statusMsg string

type model struct {
	cfg         core.Config
	data        []core.SectionData
	rate        core.RateLimit // GraphQL budget left after the last fetch
	rows        []core.Row
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
	jobs        map[string]*core.JobView // latest agent job per PR (prKey)
	spin        int
	ticking     bool
	filter      string // "/" filter; empty shows everything
	filtering   bool   // typing into the filter
	seen        core.SeenStore
	recentKey   string   // the PR whose changes are listed in recent
	recent      []string // what changed since you last looked at the selected PR
	logs        map[string]*failLog
	org         *core.Organize
	showAll     bool           // pa: show hidden and snoozed PRs too
	orgKey      bool           // p was pressed; the next key picks an organize action
	input       *textInput     // a one-line prompt in the footer (pm, pc)
	proposal    *core.Proposal // Claude's suggested reorganization, waiting for y/n
}

// textInput is a one-line prompt in the footer; enter submits, esc cancels.
type textInput struct {
	label  string
	value  string
	submit func(m *model, value string) tea.Cmd
}

func (m model) view() core.View {
	return core.View{Collapsed: m.collapsed, Filter: m.filter, Org: m.org, ShowAll: m.showAll}
}

// failLog is the tail of a failing check's log, loaded on demand with f.
type failLog struct {
	check   string
	text    string
	loading bool
}

type logMsg struct {
	key string
	log failLog
}

func prepareCmd(cfg core.Config, c core.Command, p core.PR, agentSession string) tea.Cmd {
	return func() tea.Msg {
		pp, err := core.Prepare(cfg, c, p, agentSession)
		return preparedMsg{pp, err}
	}
}

func (m *model) startCommand(c core.Command, p core.PR) tea.Cmd {
	if c.Builtin == core.BuiltinStopAgent {
		msg, err := core.StopAgent(m.jobs, p)
		if err != nil {
			msg = err.Error()
		}
		m.status, m.isErr = msg, err != nil
		return nil
	}
	job := m.jobs[core.PrKey(p.Repo, p.Number)]
	if job != nil && job.Active() && (c.Background || c.Session) {
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
		if v.Active() {
			m.ticking = true
			return jobTick()
		}
	}
	return nil
}

func (m model) runPrepared(pp core.Prepared) tea.Cmd {
	c := pp.ExecCmd()
	if pp.Cmd.Interactive {
		return tea.ExecProcess(c, func(err error) tea.Msg { return doneMsg{pp: pp, err: err} })
	}
	return func() tea.Msg {
		out, err := c.CombinedOutput()
		return doneMsg{pp, err, string(out)}
	}
}

func newModel(cfg core.Config) model {
	m := model{cfg: cfg, collapsed: map[string]bool{}, loading: true, detail: viewport.New(0, 0), jobs: core.LoadJobs(),
		seen: core.LoadSeen(), logs: map[string]*failLog{}}
	org, err := core.LoadOrganize()
	if err != nil {
		m.status, m.isErr = err.Error(), true
		org = &core.Organize{}
	}
	m.org = org
	if data, at, ok := core.LoadCache(cfg); ok {
		m.data = data
		m.status = "cached " + at.Format("15:04")
		m.rows = core.BuildRows(data, cfg, m.view())
	}
	return m
}

func fetchCmd(cfg core.Config, prev []core.SectionData) tea.Cmd {
	return func() tea.Msg {
		d, rate, err := core.FetchAll(cfg, prev)
		at := time.Now()
		if err == nil {
			_ = core.SaveCache(d, at) // shortcut: a failed cache write only costs a slower next start
		}
		return fetchedMsg{d, at, rate, err}
	}
}

func (m *model) scheduleRefresh() tea.Cmd {
	m.gen++
	if m.cfg.Refresh == 0 {
		return nil
	}
	gen := m.gen
	return tea.Tick(core.RefreshDelay(m.cfg.Refresh, m.rate, time.Now()), func(time.Time) tea.Msg { return refreshMsg(gen) })
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
		return tea.Batch(fetchCmd(m.cfg, m.data), jobTick())
	}
	return fetchCmd(m.cfg, m.data)
}

func (m model) anyRunning() bool {
	for _, v := range m.jobs {
		if v.Active() {
			return true
		}
	}
	return false
}

func (m model) listWidth() int  { return m.width * 66 / 100 }
func (m model) bodyHeight() int { return max(m.height-1, 1) }
func (m model) listRows() int   { return max(m.bodyHeight()-1, 1) } // minus the column header

func (m model) current() *core.Row {
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
	m.rows = core.BuildRows(m.data, m.cfg, m.view())
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
	m.markSeen()
	m.detail.SetContent(m.renderDetail())
	if r := m.current(); r != nil && r.Path != m.detailPath {
		m.detailPath = r.Path
		m.detail.GotoTop() // only when the selection changes, so background refreshes don't reset your scroll
	}
}

// baseline records PRs gh-prs hasn't seen before, so only later changes count as new.
func (m *model) baseline() {
	changed := false
	for _, sd := range m.data {
		for _, p := range sd.PRs {
			if k := core.PrKey(p.Repo, p.Number); m.seen != nil {
				if _, ok := m.seen[k]; !ok {
					m.seen[k] = core.SnapOf(p)
					changed = true
				}
			}
		}
	}
	if changed {
		_ = m.seen.Save() // shortcut: if it can't be saved, PRs just won't be marked new next run
	}
}

// markSeen records the selected PR as seen, keeping the list of what changed to show in the detail pane.
func (m *model) markSeen() {
	r := m.current()
	if r == nil || r.PR == nil || m.seen == nil {
		return
	}
	k, cur := core.PrKey(r.PR.Repo, r.PR.Number), core.SnapOf(*r.PR)
	if k != m.recentKey {
		m.recentKey, m.recent = k, nil
	}
	prev, ok := m.seen[k]
	if ok && prev == cur {
		return
	}
	if ok {
		m.recent = append(m.recent, core.Changes(prev, cur)...)
	}
	m.seen[k] = cur
	_ = m.seen.Save()
}

// isNew reports whether a PR changed since you last selected it.
func (m model) isNew(p core.PR) bool {
	prev, ok := m.seen[core.PrKey(p.Repo, p.Number)]
	return ok && len(core.Changes(prev, core.SnapOf(p))) > 0
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
		m.data, m.rate = msg.data, msg.rate
		m.status, m.isErr = fmt.Sprintf("updated %s · %d API points left", msg.at.Format("15:04"), m.rate.Remaining), false
		if d := core.RefreshDelay(m.cfg.Refresh, m.rate, msg.at); m.cfg.Refresh > 0 && d > m.cfg.Refresh {
			m.status, m.isErr = fmt.Sprintf("auto-refresh paused: %d GitHub API points left this hour, resumes %s (r refreshes now)",
				m.rate.Remaining, m.rate.ResetAt.Local().Format("15:04")), true
		}
		m.baseline()
		if m.org.Wake(m.data) { // snoozed PRs that changed come back
			_ = m.org.Save()
		}
		m.rebuild()
		return m, m.scheduleRefresh()
	case refreshMsg:
		if int(msg) == m.gen && !m.loading {
			m.loading = true
			return m, fetchCmd(m.cfg, m.data)
		}
	case statusMsg:
		m.status, m.isErr = string(msg), false
	case proposalMsg:
		if m.proposal == nil { // cancelled while Claude was thinking
			return m, nil
		}
		if msg.err != nil {
			m.proposal = nil
			m.status, m.isErr = msg.err.Error(), true
			m.clamp()
			return m, nil
		}
		m.proposal = msg.p
		m.detail.SetContent(m.renderDetail())
		m.detail.GotoTop()
	case logMsg:
		l := msg.log
		m.logs[msg.key] = &l
		m.detail.SetContent(m.renderDetail())
	case preparedMsg:
		if msg.err != nil {
			m.status, m.isErr = msg.pp.Cmd.Name+" failed: "+msg.err.Error(), true
			return m, nil
		}
		if msg.pp.Cmd.Background {
			v, err := core.StartJob(msg.pp)
			if err != nil {
				m.status, m.isErr = msg.pp.Cmd.Name+" failed to start: "+err.Error(), true
				return m, nil
			}
			m.jobs[core.PrKey(v.Repo, v.Number)] = v
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
				v.Update()
				finished = finished || (was == core.JobRunning && v.Status != core.JobRunning)
			}
			m.detail.SetContent(m.renderDetail())
			if finished && !m.loading { // the agent may have pushed or edited the PR: show it now
				m.loading = true
				cmds = append(cmds, fetchCmd(m.cfg, m.data))
			}
		}
		if m.anyRunning() {
			cmds = append(cmds, jobTick())
		} else {
			m.ticking = false
		}
		return m, tea.Batch(cmds...)
	case doneMsg:
		if msg.pp.Cmd.Prompt != "" && !msg.pp.Cmd.Terminal { // terminal launchers delete their own prompt file
			for _, e := range msg.pp.Env {
				if f, ok := strings.CutPrefix(e, "PR_PROMPT_FILE="); ok {
					os.Remove(f)
				}
			}
		}
		label := fmt.Sprintf("%s #%d", msg.pp.Cmd.Name, msg.pp.Pr.Number)
		if msg.err != nil {
			detail := core.LastLine(msg.out)
			if detail == "" {
				detail = msg.err.Error()
			}
			m.status, m.isErr = "✗ "+label+": "+detail, true
			return m, nil
		}
		m.status, m.isErr = "✓ "+label, false
		if msg.pp.Cmd.Refresh && !m.loading {
			m.loading = true
			return m, fetchCmd(m.cfg, m.data)
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
		{"/", "filter (enter keeps it, esc clears)"}, {"f", "load the failing check's log"},
		{"o", "open in browser"}, {"i", "open the Linear issue"}, {"r", "refresh"}, {"q", "quit"},
		{"pp ps ph", "pin, snooze until it changes, hide (again to undo)"},
		{"pm", "move to a project (empty: back to its label)"},
		{"pa", "show hidden and snoozed PRs"}, {"pc", "ask Claude to reorganize"},
		{"★ ● ○ ◆", "pinned · your turn · their turn · changed since you looked"},
	} {
		fmt.Fprintf(&b, "%-14s %s\n", kv[0], kv[1])
	}
	b.WriteString("\n" + bold.Render("Commands") + "  " + dim.Render(core.ConfigPath()) + "\n\n")
	for _, c := range m.cfg.Commands {
		fmt.Fprintf(&b, "%-14s %s\n", c.Key, c.Name)
	}
	return b.String()
}

func (m model) footer() string {
	if c := m.confirming; c != nil {
		return yellow.Bold(true).Render(fmt.Sprintf("%s #%d? y/N", c.cmd.Name, c.pr.Number))
	}
	help := "j/k move · / filter · e details · o browser · cc agent · ct take over · ? keys · q quit"
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
	if m.filter != "" {
		help = "filter: " + m.filter + " (/ edit · esc clear) · " + help
	}
	if m.filtering {
		help = "/" + m.filter + "█  enter keep · esc clear"
	}
	if m.orgKey {
		help = "p…  p pin · s snooze · m move to project · h hide · a show all · c ask Claude"
	}
	if m.input != nil {
		help = m.input.label + m.input.value + "█  enter ok · esc cancel"
	}
	if m.proposal != nil && !m.proposal.Loading {
		help = "Claude's proposal: y apply · n discard · j/k scroll"
	}
	if m.showAll {
		help = "showing hidden & snoozed (pa to stop) · " + help
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
