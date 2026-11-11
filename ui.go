package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/cli/go-gh/v2/pkg/browser"
)

var (
	plain        = lipgloss.NewStyle()
	dim          = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	bold         = lipgloss.NewStyle().Bold(true)
	green        = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	red          = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	yellow       = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	cyan         = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	sectionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))
	selBg        = lipgloss.Color("238")
	stripeBg     = lipgloss.Color("235")

	// Solid badges mark the reasons a PR needs action.
	badgeFail     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("1"))
	badgeChanges  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("208"))
	badgeConflict = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("5"))

	statusStyles = map[string]lipgloss.Style{
		NeedsAction.String():   bold.Foreground(lipgloss.Color("1")),
		WaitingReview.String(): bold.Foreground(lipgloss.Color("3")),
		ReadyToMerge.String():  bold.Foreground(lipgloss.Color("2")),
		Draft.String():         bold.Foreground(lipgloss.Color("8")),
	}
)

// Widths of the right-aligned table columns.
const (
	colLines   = 13
	colCI      = 8
	colReview  = 11
	colThreads = 6
	colMerge   = 10
	colAgent   = 18
)

var spinner = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

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

// shaded reports whether row i is an every-other PR row in its group, restarting after each header.
func (m model) shaded(i int) bool {
	n := 0
	for ; i >= 0 && m.rows[i].PR != nil; i-- {
		n++
	}
	return n > 0 && n%2 == 0
}

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

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	if k == "ctrl+c" {
		return m, tea.Quit
	}
	r := m.current()

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

type seg struct {
	text  string
	style lipgloss.Style
}

func spaces(n int) string { return strings.Repeat(" ", max(n, 0)) }

// compactCount keeps the lines column a fixed width: 47981 → "48k". The detail pane shows exact counts.
func compactCount(n int) string {
	if n < 10000 {
		return fmt.Sprint(n)
	}
	return fmt.Sprintf("%dk", (n+500)/1000)
}

func columnHeader(w int) string {
	right := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %-*s  %-*s", colLines, "lines", colCI-1, "ci", colReview-1, "review", colThreads-1, "unres", colMerge-1, "merge", colAgent-1, "agent")
	return dim.Underline(true).Render(" pr" + spaces(w-3-lipgloss.Width(right)) + right)
}

// agentCell is the agent column: a spinner while running, then the outcome.
func agentCell(v *jobView, spin int) (string, lipgloss.Style) {
	if v == nil {
		return "", dim
	}
	switch v.Status {
	case JobRunning:
		return fmt.Sprintf(" %s %s %s", spinner[spin%len(spinner)], v.Name, shortDuration(v.elapsed())), yellow
	case JobDone:
		return " ✓ " + v.Name, green
	}
	return " ✗ " + v.Name + " ", badgeFail
}

// prColumns renders the right-aligned table cells for a PR.
func prColumns(p PR, job *jobView, spin int) []seg {
	var out []seg
	col := func(text string, st lipgloss.Style, w int) {
		out = append(out, seg{" ", plain}, seg{text, st}, seg{spaces(w - lipgloss.Width(text)), plain})
	}

	add, del := "+"+compactCount(p.Additions), "−"+compactCount(p.Deletions)
	out = append(out, seg{add, green}, seg{" ", plain}, seg{del, red}, seg{spaces(colLines - lipgloss.Width(add) - 1 - lipgloss.Width(del)), plain})

	switch p.CI {
	case "FAILURE", "ERROR":
		col(" ✗ fail ", badgeFail, colCI)
	case "PENDING", "EXPECTED":
		col(" • run", yellow, colCI)
	case "SUCCESS":
		col(" ✓ pass", green, colCI)
	default:
		col(" –", dim, colCI)
	}

	switch p.ReviewDecision {
	case "CHANGES_REQUESTED":
		col(" changes ", badgeChanges, colReview)
	case "APPROVED":
		col(" ✓ approved", green, colReview)
	case "REVIEW_REQUIRED":
		col(" waiting", dim, colReview)
	default:
		col(" –", dim, colReview)
	}

	if p.Unresolved > 0 {
		col(fmt.Sprintf(" %d", p.Unresolved), yellow, colThreads)
	} else {
		col("", dim, colThreads)
	}

	if p.Mergeable == "CONFLICTING" {
		col(" conflict ", badgeConflict, colMerge)
	} else {
		col("", dim, colMerge)
	}

	text, st := agentCell(job, spin)
	col(ansi.Truncate(text, colAgent, "…"), st, colAgent)
	return out
}

func (m model) renderRow(r Row, isSel, shade bool, w int) string {
	indent := strings.Repeat("  ", r.Depth)
	var left seg
	var right []seg

	if r.PR == nil {
		arrow := "▾"
		if m.collapsed[r.Path] {
			arrow = "▸"
		}
		st, ok := statusStyles[r.Title]
		if !ok {
			st = bold
		}
		if r.Depth == 0 {
			st = sectionStyle
		}
		left = seg{fmt.Sprintf("%s%s %s (%d)", indent, arrow, r.Title, r.Count), st}
	} else {
		p := r.PR
		stack, on := "", ""
		if r.Stack > 0 {
			stack = strings.Repeat("  ", r.Stack-1) + "↳ "
		} else if r.StackedOn > 0 { // parent is in another group
			stack, on = "↳ ", fmt.Sprintf("(on #%d) ", r.StackedOn)
		}
		left = seg{fmt.Sprintf("%s%s%-7s%s%s", indent, stack, fmt.Sprintf("#%d", p.Number), on, p.Title), plain}
		if isSel {
			left.style = bold
		}
		right = prColumns(*p, m.jobs[prKey(p.Repo, p.Number)], m.spin)
	}

	rightW := 0
	for _, s := range right {
		rightW += lipgloss.Width(s.text)
	}
	left.text = ansi.Truncate(left.text, max(w-rightW-1, 0), "…")
	segs := append([]seg{left, {spaces(w - lipgloss.Width(left.text) - rightW), plain}}, right...)

	var b strings.Builder
	for _, s := range segs {
		st := s.style
		if _, none := st.GetBackground().(lipgloss.NoColor); none { // badges keep their own color
			if isSel {
				st = st.Background(selBg)
			} else if shade {
				st = st.Background(stripeBg)
			}
		}
		b.WriteString(st.Render(s.text))
	}
	return b.String()
}

func ciText(state string) string {
	if state == "" {
		return "no checks"
	}
	return strings.ToLower(state)
}

func reviewText(d string) string {
	switch d {
	case "APPROVED":
		return "approved"
	case "CHANGES_REQUESTED":
		return "changes requested"
	case "REVIEW_REQUIRED":
		return "review required"
	}
	return "none required"
}

// agentSummary is the block agents are asked to end with (see prompt_context).
type agentSummary struct {
	Headline string
	Changes  []string
	Pushed   string
	NeedsYou string
}

// parseSummary reads the last DONE: block of an agent's final message; ok is false if there is none.
func parseSummary(result string) (agentSummary, bool) {
	var s agentSummary
	lines := strings.Split(result, "\n")
	start := -1
	for i, l := range lines {
		if strings.HasPrefix(cleanLine(l), "DONE:") {
			start = i
		}
	}
	if start < 0 {
		return s, false
	}
	for _, l := range lines[start:] {
		l = cleanLine(l)
		switch {
		case strings.HasPrefix(l, "DONE:"):
			s.Headline = strings.TrimSpace(strings.TrimPrefix(l, "DONE:"))
		case strings.HasPrefix(l, "- "), strings.HasPrefix(l, "* "):
			s.Changes = append(s.Changes, strings.TrimSpace(l[2:]))
		case strings.HasPrefix(l, "PUSHED:"):
			s.Pushed = strings.TrimSpace(strings.TrimPrefix(l, "PUSHED:"))
		case strings.HasPrefix(l, "NEEDS YOU:"):
			s.NeedsYou = strings.TrimSpace(strings.TrimPrefix(l, "NEEDS YOU:"))
		}
	}
	return s, s.Headline != ""
}

// cleanLine drops markdown emphasis and code fences agents sometimes wrap the block in.
func cleanLine(l string) string {
	l = strings.TrimSpace(strings.ReplaceAll(l, "**", ""))
	return strings.TrimSpace(strings.TrimLeft(l, "#>"))
}

var codeSpan = regexp.MustCompile("`[^`]+`")

// colorSpans styles s, showing `code spans` (usually file names) in cyan without the backticks.
func colorSpans(s string, st lipgloss.Style) string {
	var b strings.Builder
	last := 0
	for _, loc := range codeSpan.FindAllStringIndex(s, -1) {
		b.WriteString(st.Render(s[last:loc[0]]) + cyan.Render(s[loc[0]+1:loc[1]-1]))
		last = loc[1]
	}
	b.WriteString(st.Render(s[last:]))
	return b.String()
}

// fit cuts a line to width, for one-line entries like the running agent's recent actions.
func fit(s string, width int, st lipgloss.Style) string {
	return colorSpans(ansi.Truncate(s, width, "…"), st)
}

// wrapped word-wraps s to width after prefix, indenting continuation lines under the text.
func wrapped(prefix, s string, width int, st lipgloss.Style) []string {
	pw := lipgloss.Width(prefix)
	body := lipgloss.NewStyle().Width(max(width-pw, 10)).Render(colorSpans(s, st))
	lines := strings.Split(body, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " ")
		if i == 0 {
			lines[i] = st.Render(prefix) + lines[i]
		} else {
			lines[i] = spaces(pw) + lines[i]
		}
	}
	return lines
}

const maxChanges = 10

// renderJob draws the agent box: what the agent did, wrapped to the pane width. It can grow
// taller than the pane, which scrolls (e, then j/k).
func renderJob(v *jobView, width int) string {
	inner := max(width-4, 10) // border + padding on each side
	var lines []string
	add := func(s string, st lipgloss.Style) { lines = append(lines, fit(s, inner, st)) }
	wrap := func(prefix, s string, st lipgloss.Style) { lines = append(lines, wrapped(prefix, s, inner, st)...) }

	title, border := "Agent · "+v.Name, lipgloss.Color("3")
	switch v.Status {
	case JobRunning:
		add(title+" · running "+shortDuration(v.elapsed()), yellow.Bold(true))
		acts := v.Activity
		if len(acts) > 4 {
			acts = acts[len(acts)-4:]
		}
		for _, a := range acts {
			add("› "+a, dim)
		}
		if len(acts) == 0 {
			add("starting…", dim)
		}
	case JobDone, JobFailed:
		if v.Status == JobDone {
			add(title+" ✓", green.Bold(true))
			border = lipgloss.Color("2")
		} else {
			add(title+" ✗", red.Bold(true))
			border = lipgloss.Color("1")
		}
		if sum, ok := parseSummary(v.Result); ok {
			wrap("", sum.Headline, bold)
			for i, c := range sum.Changes {
				if i == maxChanges {
					add(fmt.Sprintf("  …and %d more", len(sum.Changes)-maxChanges), dim)
					break
				}
				wrap("+ ", c, green)
			}
			switch p := sum.Pushed; {
			case strings.HasPrefix(strings.ToUpper(p), "NO"):
				wrap("not pushed: ", strings.TrimSpace(strings.TrimLeft(p[2:], ":")), yellow)
			case p != "":
				wrap("↑ pushed ", p, green)
			}
			if n := sum.NeedsYou; n != "" && !strings.EqualFold(strings.Trim(n, ". "), "nothing") {
				wrap("! needs you: ", n, yellow.Bold(true))
			}
		} else { // no summary block (an older prompt, the deploy script, an error)
			for _, l := range strings.Split(strings.TrimSpace(v.Result), "\n") {
				wrap("", l, plain)
			}
		}
		meta := "after " + shortDuration(v.elapsed())
		if v.Status == JobDone {
			meta = "done in " + shortDuration(v.elapsed())
		}
		if v.Turns > 0 {
			meta += fmt.Sprintf(" · %d turns", v.Turns)
		}
		if v.Cost > 0 {
			meta += fmt.Sprintf(" · $%.2f", v.Cost)
		}
		wrap("", meta+" · ct take over", dim)
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(border).
		Padding(0, 1).Width(inner + 2).Render(strings.Join(lines, "\n"))
}

func (m model) renderDetail() string {
	r := m.current()
	w := max(m.detail.Width, 1)
	wrap := lipgloss.NewStyle().Width(w)
	if m.showHelp {
		return wrap.Render(m.helpText())
	}
	if r == nil {
		return ""
	}
	if r.PR == nil {
		return wrap.Render(bold.Render(r.Title) + "\n\n" + fmt.Sprintf("%d PRs", r.Count))
	}
	p := r.PR
	var b strings.Builder
	b.WriteString(bold.Render(p.Title) + "\n")
	b.WriteString(dim.Render(fmt.Sprintf("%s#%d · @%s", p.Repo, p.Number, p.Author)) + "\n")
	branches := fmt.Sprintf("%s → %s", p.HeadRefName, p.BaseRefName)
	if r.StackedOn > 0 {
		branches += fmt.Sprintf(" (stacked on #%d)", r.StackedOn)
	}
	b.WriteString(dim.Render(branches) + "\n\n")
	fmt.Fprintf(&b, "%s %s  %d files\n", green.Render(fmt.Sprintf("+%d", p.Additions)), red.Render(fmt.Sprintf("−%d", p.Deletions)), p.ChangedFiles)
	fmt.Fprintf(&b, "%s · CI %s · review %s · %d unresolved\n", bold.Render(statusOf(*p).String()), ciText(p.CI), reviewText(p.ReviewDecision), p.Unresolved)
	if p.Mergeable == "CONFLICTING" {
		b.WriteString(red.Render("Merge conflict") + "\n")
	}
	if len(p.Labels) > 0 {
		b.WriteString(dim.Render("labels: "+strings.Join(p.Labels, ", ")) + "\n")
	}
	if v := m.jobs[prKey(p.Repo, p.Number)]; v != nil {
		b.WriteString("\n" + renderJob(v, w) + "\n")
	}
	body := strings.TrimSpace(p.Body)
	if body == "" {
		body = dim.Render("(no description)")
	}
	b.WriteString("\n" + body)
	return wrap.Render(b.String())
}
