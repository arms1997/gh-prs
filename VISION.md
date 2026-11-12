# gh-prs — Product Vision

_Living document. Update it when the product or the plan changes._

## Problem

Engineers working on several projects at once lose track of their PRs. GitHub's lists are flat: they
don't show which project a PR belongs to, what's blocking it, or whose turn it is. Tools like gh-dash
help, but their sections can't nest, and they can't be reorganized just by asking.

## Vision

A dashboard that answers **"what should I do next?"** across all my PRs, organized the way I think
about my work (by project, then by action), and that hands a PR to Claude with the right context
when I want help, then lets me talk to that agent until the PR merges.

It started as a terminal UI and now has two front ends on one core: the terminal UI for speed, and
a macOS app for a richer view (what's left before merge, checks, files, the conversation with
Claude) when that helps. Both read the same config, cache, organize file and agent jobs, so they
can be used side by side.

## Users

- **Now:** me, someone working across several repos and projects at once, with stacked PRs.
- **Next:** the Augment engineering team, installed as a `gh` extension.

## Principles

1. **Answer "what's next" at a glance.** The reasons a PR needs action (failing CI, changes requested, a conflict) should pop out, in color.
2. **Organized your way.** Grouping is declarative config (labels, projects, status, repo) that the user or Claude can edit.
3. **Keyboard first.** Vim keys and arrow keys; every action is a keystroke away.
4. **Never wait.** Show cached data instantly and refresh in the background.
5. **Read-only by default.** Anything that changes GitHub is an explicit action the user triggers.
6. **Safe with untrusted text.** PR content written by others is sanitized before it's rendered or passed to tools.
7. **Small and dependency-light.** A single Go binary that reuses the `gh` login, with no new tokens to manage.
8. **One core, two front ends.** Behaviour lives in `core/`; the terminal UI and the desktop app only present it, so a feature works the same in both.
9. **Agents are collaborators, not fire-and-forget.** You can see what an agent did, answer its questions, reply to it, and stop it, from wherever you are.

## Non-goals

- Replacing the GitHub web UI for full code review.
- Managing issues or project boards.
- Hosting a server or storing data anywhere but locally.

## What exists today

### Shared core (both front ends)

- Sections defined by GitHub search queries, with nested `group_by` (project, status, repo, turn)
- Action buckets: Needs action, Waiting for review, Ready to merge, Draft
- Stacked PRs indented under their parent, or marked `↳ (on #N)` when the parent is in another group
- Disk cache, instant start, background refresh; page sizes follow last refresh, and refresh pauses when the GitHub API budget runs low
- Every check on the latest commit (pass, fail, running, skipped), at no extra API cost
- Configurable key-sequence commands (PR data passed as `$PR_*` env vars, never interpolated); commands can ask for a line of text first (`input`), and a few are built in (`builtin: stop_agent`)
- Organizing (`p…`): pin, snooze until it changes, move to a project, hide, all local in `organize.yml`; `pc` asks Claude to reorganize, previewed as a diff
- Awareness: whose turn it is and why, what changed since you looked, failing checks, Linear links, filter

### Agents

- Background agents run headless Claude (Sonnet) in the PR's worktree, detached so they outlive gh-prs: `cc` get it mergeable, `cf` fix CI, `ca` address comments, `cr` rebase
- `cm` manage: an agent on Claude's `/loop` keeps a PR mergeable every 15 minutes until it merges (8h / $10 cap)
- Conversations: agents end with a summary (what they did, what they pushed, what they need from you, and suggested answers). `cy` replies by resuming the agent's own session, stopping a running loop first; `ci` asks Claude about a PR in a new session; `ct` takes the session over in Claude Code
- `cx` stops an agent and everything it started
- Safety: agents push only to your own PRs (a git push-URL rewrite blocks the rest), refuse dirty worktrees, and have time limits enforced by a watchdog in their process group
- `D` merge + deploy prod with deploy-window, approval and scope checks, then Claude/Datadog health checks; `V` opens a revert PR

### Terminal UI

- Color-coded table: lines changed, CI, review, unresolved threads, merge conflict, agent column with a live spinner
- Detail pane with the description, the agent box (recent steps while running, then its summary), failing check logs (`f`)
- Vim navigation, collapsible groups, `?` key list, pluggable openers (Ghostty, tmux, herdr, …)

### Desktop app (macOS, Wails)

- Follows the system light/dark theme, with Augment's design system colors
- Main view: quick filters with counts (waiting for review, ready to merge, draft, CI failing, conflicts, agent running), a "needs attention" column and a suggested **next step** per PR that ⏎ runs
- Detail pane tabs: **Overview** (what's left before merge, with a progress bar; the PR's `- [ ]` checklist; the description), **Claude**, **Checks** (grouped by outcome), **Files** (loaded on demand)
- Claude tab: the agent's status, its open question with answer buttons and a reply box, the last result, each round of a manage loop, and its recent steps; a message box (`i`) under every PR
- Claude sessions page (`ga`): every agent run across your PRs, with open, stop, take over and run again
- ⌘K command palette that shows each action's shortcut; the same keys as the terminal UI, which stay out of the way while you type
- Front end split into pages, features and shared components; unit tests with Vitest

## Roadmap

| Phase | Theme | Scope | Status |
|---|---|---|---|
| 1 | Core | Fetch, nested grouping, navigation, detail pane, table, stacking, cache | ✅ Done |
| 2 | Actions | Configurable key→command bindings; `cc` opens Claude with prompt templates (fix CI, address comments, rebase, write description); checkout into a worktree; mark ready, merge, label, copy | ✅ Done |
| 2.5 | Background agents | Headless agent jobs, agent column, live activity, take over with `ct` | ✅ Done |
| 3 | Awareness | Whose turn it is ("on me" vs "on them"), changes since last viewed, failing check name and logs, Linear link, `/` filter | ✅ Done |
| 4 | Customization | Pins and overrides kept between sessions, Claude editing the grouping rules with a preview of the change | ✅ Done |
| 5 | Team | Agentic onboarding (Claude Code plugin: "set up gh-prs for me"), publish as a `gh` extension, neutral defaults | In progress |
| 6 | Desktop | `core/` package shared by both front ends; Wails app redesigned on Augment's design system: quick filters, next step, before-merge checklist, checks and files tabs, ⌘K palette. Next: `pc`, check logs, "changed since you looked", signing | In progress |
| 7 | Conversations | Talk to agents: their questions with answers, reply by resuming the session (`cy`), ask in a new one (`ci`), stop (`cx`), all sessions in one place | ✅ Done |
| 8 | Notifications | An inbox of what changed (Claude needs you, ready to merge, checks failing, new conflict, review), per-event settings, macOS alerts | Next |
| 9 | Factory | How gh-prs itself is built: tests at every layer, CI gates, review, releases | Next |

## Open questions

- Should a stacked child always render under its parent, even when their statuses differ?
- PRs from forks: worktrees only work for branches pushed to the main repo.
- Human escalation for background agents: questions now surface as "needs you" with answers; a denied tool still only shows in the agent's steps.
- Does a manage loop reliably carry on its rounds after a reply resumes its session? Expected, not yet seen.
- macOS alerts with answer buttons need a signed app and native notification code (Wails v2 has none).
- Deploy health checks cost about $1.20 each ($5 per deploy): fewer checks or a cheaper model?
- Should repos that don't require reviews count an approval as "Ready to merge"?
- Team rollout: one shared default config, or per-team presets?

## Changelog

- **2026-11-12:** Unit tests for the desktop app's logic (filters, next step, before-merge, Markdown sanitizing) with Vitest.
- **2026-11-12:** Phase 7 shipped: conversations with agents. Answer an agent's question, reply by resuming its session (`cy`), ask in a new one (`ci`), stop it (`cx`); Claude tab and sessions page in the app.
- **2026-11-12:** Desktop app redesign on Augment's design system: main view with quick filters and next step, detail tabs (overview with before-merge checklist, Claude, checks, files), ⌘K palette. Front end split into pages and components.
- **2026-11-12:** Desktop app (phase 6): the logic moved into `core/`, shared by the terminal UI and a Wails macOS app that follows the system theme.
- **2026-11-12:** GitHub API budget: page sizes follow the last refresh, and refresh pauses until the hourly reset when points run low.
- **2026-11-12:** Agentic onboarding: a Claude Code plugin ("set up gh-prs for me") interviews you and writes your config.

- **2026-11-11:** Phase 4 shipped: pins, snoozes, local projects, hidden PRs, and Claude reorganizing with a diff preview.
- **2026-11-11:** Phase 3 shipped: whose turn, what changed, failing check logs, Linear links, filter.
- **2026-11-11:** Agent box: bounded summary of what an agent changed.
- **2026-11-11:** Time limits for background jobs.
- **2026-11-11:** Dropped `cd` (describe): it rewrote PR descriptions, and PRs are already written with one.
- **2026-11-11:** `D` merge + deploy prod with Datadog health checks, `V` revert PR.
- **2026-11-11:** Background agents: headless Claude jobs with an agent column, push on completion, `ct` to take over.
- **2026-11-12:** `cm` manage: a background agent on Claude's `/loop` keeps a PR mergeable until it merges. Agents push only to your own PRs; on others' they commit and leave the push to you. Agents refuse dirty worktrees and use Sonnet. (`dontAsk` was tried and dropped: it refused plain git commit/push chains.)
- **2026-11-11:** Phase 2 shipped: commands, Claude launchers, worktrees. `cc` opens a new terminal window, configurable via `terminal`.
- **2026-11-11:** Phase 1 shipped: grouping, table, stacking, cache and background refresh.
