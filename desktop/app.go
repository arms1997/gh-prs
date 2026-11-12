package main

// The desktop app's backend: the same core as the terminal UI, exposed as methods the web
// front end calls (Wails binds every exported method of App).

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"gh-prs/core"
)

type App struct {
	ctx context.Context

	mu      sync.Mutex
	cfg     core.Config
	data    []core.SectionData
	rate    core.RateLimit
	jobs    map[string]*core.JobView
	org     *core.Organize
	loading bool
	status  string
	isErr   bool
	kick    chan struct{} // wakes the refresh loop for an immediate refresh
}

func NewApp(cfg core.Config) *App {
	a := &App{cfg: cfg, jobs: core.LoadJobs(), kick: make(chan struct{}, 1)}
	org, err := core.LoadOrganize()
	if err != nil {
		a.status, a.isErr = err.Error(), true
		org = &core.Organize{}
	}
	a.org = org
	if data, at, ok := core.LoadCache(cfg); ok {
		a.data, a.status = data, "cached "+at.Format("15:04")
	}
	return a
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go a.refreshLoop()
	go a.jobLoop()
}

// refreshLoop fetches now, then on the configured interval, waiting for the hourly reset when
// the GitHub API budget runs low, or right away when kicked.
func (a *App) refreshLoop() {
	for {
		a.fetch()
		a.mu.Lock()
		every := a.cfg.Refresh
		if every == 0 {
			every = 24 * time.Hour // auto-refresh off: only kicks
		}
		wait := core.RefreshDelay(every, a.rate, time.Now())
		a.mu.Unlock()
		select {
		case <-time.After(wait):
		case <-a.kick:
		}
	}
}

func (a *App) fetch() {
	a.mu.Lock()
	cfg, prev := a.cfg, a.data
	a.loading = true
	a.mu.Unlock()

	data, rate, err := core.FetchAll(cfg, prev)
	at := time.Now()

	a.mu.Lock()
	defer a.mu.Unlock()
	a.loading = false
	if err != nil {
		a.status, a.isErr = "refresh failed, showing older data: "+err.Error(), true
		return
	}
	_ = core.SaveCache(data, at) // shortcut: a failed cache write only costs a slower next start
	a.data, a.rate = data, rate
	a.status, a.isErr = fmt.Sprintf("updated %s · %d API points left", at.Format("15:04"), rate.Remaining), false
	if d := core.RefreshDelay(cfg.Refresh, rate, at); cfg.Refresh > 0 && d > cfg.Refresh {
		a.status, a.isErr = fmt.Sprintf("auto-refresh paused: %d GitHub API points left this hour, resumes %s",
			rate.Remaining, rate.ResetAt.Local().Format("15:04")), true
	}
	if a.org.Wake(data) { // snoozed PRs that changed come back
		_ = a.org.Save()
	}
}

// jobLoop follows agent logs, and refreshes PRs when an agent finishes a run (it may have pushed).
func (a *App) jobLoop() {
	for range time.Tick(time.Second) {
		a.mu.Lock()
		finished := false
		for _, v := range a.jobs {
			was := v.Status
			v.Update()
			finished = finished || (was == core.JobRunning && v.Status != core.JobRunning)
		}
		a.mu.Unlock()
		if finished {
			a.Refresh()
		}
	}
}

// Refresh fetches PRs now.
func (a *App) Refresh() {
	select {
	case a.kick <- struct{}{}:
	default: // one is already queued
	}
}

// ---- state for the front end ----

type PRView struct {
	core.PR
	Key       string
	Status    string
	OnMe      bool
	TurnWhy   string
	Project   string
	Pinned    bool
	Hidden    bool
	Snoozed   bool
	LinearURL string
}

type RowView struct {
	Title     string
	Depth     int
	Count     int
	Path      string
	Stack     int
	StackedOn int
	PR        *PRView
}

type JobState struct {
	Name     string
	Key      string
	State    string // running, waiting, done, failed
	Activity []string
	Result   string
	Summary  *core.AgentSummary
	Cost     float64
	Turns    int
	Elapsed  string
}

type CommandView struct {
	Key        string
	Name       string
	Background bool
	Confirm    bool
}

type State struct {
	Rows     []RowView
	Jobs     map[string]JobState
	Commands []CommandView
	Status   string
	IsErr    bool
	Loading  bool

	// Unused: makes Wails' TypeScript generator emit core.Check, which it skips when it's
	// only reachable through the embedded core.PR.
	CheckType core.Check
}

// State returns the grouped PR list and agent jobs. collapsed holds the Paths of collapsed groups.
func (a *App) State(filter string, collapsed []string, showAll bool) State {
	a.mu.Lock()
	defer a.mu.Unlock()
	col := map[string]bool{}
	for _, p := range collapsed {
		col[p] = true
	}
	rows := core.BuildRows(a.data, a.cfg, core.View{Collapsed: col, Filter: filter, Org: a.org, ShowAll: showAll})
	s := State{Jobs: map[string]JobState{}, Status: a.status, IsErr: a.isErr, Loading: a.loading}
	for _, r := range rows {
		rv := RowView{Title: r.Title, Depth: r.Depth, Count: r.Count, Path: r.Path, Stack: r.Stack, StackedOn: r.StackedOn}
		if p := r.PR; p != nil {
			onMe, why := core.TurnOf(*p)
			rv.PR = &PRView{PR: *p, Key: core.PrKey(p.Repo, p.Number), Status: core.StatusOf(*p).String(), OnMe: onMe, TurnWhy: why,
				Project: p.Project, Pinned: p.Pinned, Hidden: p.Hidden, Snoozed: p.Snoozed}
			if p.Linear != "" && a.cfg.LinearWorkspace != "" {
				rv.PR.LinearURL = core.LinearURL(a.cfg.LinearWorkspace, p.Linear)
			}
		}
		s.Rows = append(s.Rows, rv)
	}
	for k, v := range a.jobs {
		js := JobState{Name: v.Name, Key: v.Key, Activity: v.Activity, Result: v.Result, Cost: v.Cost, Turns: v.Turns, Elapsed: core.ShortDuration(v.Elapsed())}
		switch {
		case v.Waiting:
			js.State = "waiting"
		case v.Status == core.JobRunning:
			js.State = "running"
		case v.Status == core.JobDone:
			js.State = "done"
		default:
			js.State = "failed"
		}
		if sum, ok := core.ParseSummary(v.Result); ok {
			js.Summary = &sum
		}
		s.Jobs[k] = js
	}
	for _, c := range a.cfg.Commands {
		s.Commands = append(s.Commands, CommandView{Key: c.Key, Name: c.Name, Background: c.Background, Confirm: c.Confirm})
	}
	return s
}

func (a *App) find(key string) (core.PR, bool) {
	for _, sd := range a.data {
		for _, p := range sd.PRs {
			if core.PrKey(p.Repo, p.Number) == key {
				return p, true
			}
		}
	}
	return core.PR{}, false
}

// ---- actions ----

// Run runs the command bound to cmdKey on a PR and returns a status line.
func (a *App) Run(cmdKey, prKey string) (string, error) {
	a.mu.Lock()
	p, ok := a.find(prKey)
	var c core.Command
	for _, cc := range a.cfg.Commands {
		if cc.Key == cmdKey {
			c = cc
		}
	}
	job := a.jobs[prKey]
	cfg := a.cfg
	a.mu.Unlock()
	if !ok || c.Key == "" {
		return "", fmt.Errorf("unknown PR or command")
	}
	if job != nil && job.Active() && (c.Background || c.Session) {
		// One agent per PR: they'd share the worktree, and a session can't be resumed while it runs.
		return "", fmt.Errorf("an agent is still running on #%d (%s)", p.Number, job.Name)
	}
	if c.Confirm {
		answer, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{Type: runtime.QuestionDialog,
			Title: c.Name, Message: fmt.Sprintf("%s #%d (%s)?", c.Name, p.Number, p.Repo)})
		if err != nil || answer != "Yes" {
			return "cancelled", nil
		}
	}
	if c.Interactive { // there's no terminal to hand over here: open it in your opener instead
		c.Interactive, c.Terminal = false, true
	}
	session := ""
	if job != nil {
		session = job.ID
	}
	pp, err := core.Prepare(cfg, c, p, session)
	if err != nil {
		return "", fmt.Errorf("%s failed: %w", c.Name, err)
	}
	if c.Background {
		v, err := core.StartJob(pp)
		if err != nil {
			return "", fmt.Errorf("%s failed to start: %w", c.Name, err)
		}
		a.mu.Lock()
		a.jobs[prKey] = v
		a.mu.Unlock()
		return fmt.Sprintf("agent started: %s #%d", v.Name, v.Number), nil
	}
	go func() {
		out, err := pp.ExecCmd().CombinedOutput()
		if c.Prompt != "" && !c.Terminal { // terminal launchers delete their own prompt file
			for _, e := range pp.Env {
				if f, ok := strings.CutPrefix(e, "PR_PROMPT_FILE="); ok {
					os.Remove(f)
				}
			}
		}
		label := fmt.Sprintf("%s #%d", c.Name, p.Number)
		a.mu.Lock()
		if err != nil {
			detail := core.LastLine(string(out))
			if detail == "" {
				detail = err.Error()
			}
			a.status, a.isErr = "✗ "+label+": "+detail, true
		} else {
			a.status, a.isErr = "✓ "+label, false
		}
		a.mu.Unlock()
		if err == nil && c.Refresh {
			a.Refresh()
		}
	}()
	return c.Name + " #" + fmt.Sprint(p.Number) + "…", nil
}

// OpenURL opens a web link in the browser. Only http(s) links: PR text is untrusted.
func (a *App) OpenURL(link string) error {
	u, err := url.Parse(link)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return fmt.Errorf("not a web link: %q", link)
	}
	runtime.BrowserOpenURL(a.ctx, u.String())
	return nil
}

// Organize toggles pin, hide or snooze on a PR ("pin", "hide", "snooze"), or moves it to a
// project ("move", with the project name in arg; empty clears it).
func (a *App) Organize(action, prKey, arg string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	p, ok := a.find(prKey)
	if !ok {
		return "", fmt.Errorf("unknown PR")
	}
	msg := ""
	switch action {
	case "pin":
		msg = map[bool]string{true: "pinned", false: "unpinned"}[core.Toggle(&a.org.Pinned, prKey)]
	case "hide":
		msg = map[bool]string{true: "hidden", false: "shown again"}[core.Toggle(&a.org.Hidden, prKey)]
	case "snooze":
		if a.org.IsSnoozed(prKey) {
			delete(a.org.Snoozed, prKey)
			msg = "unsnoozed"
		} else {
			a.org.Snooze(p)
			msg = "snoozed until it changes"
		}
	case "move":
		a.org.SetProject(prKey, strings.TrimSpace(arg))
		msg = "moved to " + map[bool]string{true: "its label's project", false: strings.TrimSpace(arg)}[strings.TrimSpace(arg) == ""]
	default:
		return "", fmt.Errorf("unknown action %q", action)
	}
	if err := a.org.Save(); err != nil {
		return "", err
	}
	return fmt.Sprintf("#%d %s", p.Number, msg), nil
}

// Projects lists project names in use, for the move picker.
func (a *App) Projects() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	var out []string
	for _, sd := range a.data {
		for _, p := range sd.PRs {
			for _, l := range p.Labels {
				if name, ok := strings.CutPrefix(l, a.cfg.ProjectLabelPrefix); ok && a.cfg.ProjectLabelPrefix != "" && !slices.Contains(out, name) {
					out = append(out, name)
				}
			}
		}
	}
	for _, name := range a.org.Projects {
		if !slices.Contains(out, name) {
			out = append(out, name)
		}
	}
	slices.Sort(out)
	return out
}
