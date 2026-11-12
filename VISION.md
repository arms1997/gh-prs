# gh-prs — Product Vision

_Living document. Update it when the product or the plan changes._

## Problem

Engineers working on several projects at once lose track of their PRs. GitHub's lists are flat: they
don't show which project a PR belongs to, what's blocking it, or whose turn it is. Tools like gh-dash
help, but their sections can't nest, and they can't be reorganized just by asking.

## Vision

A terminal dashboard that answers **"what should I do next?"** across all my PRs, organized the way
I think about my work (by project, then by action), and that hands a PR to Claude with the right
context when I want help fixing it.

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

## Non-goals

- Replacing the GitHub web UI for full code review.
- Managing issues or project boards.
- Hosting a server or storing data anywhere but locally.

## What exists today

- Sections defined by GitHub search queries, with nested `group_by` (project, status, repo)
- Action buckets: Needs action, Waiting for review, Ready to merge, Draft
- Color-coded table: lines changed, CI, review, unresolved threads, merge conflict
- Stacked PRs indented under their parent, or marked `↳ (on #N)` when the parent is in another group
- Detail pane: description, branches, stats
- Vim navigation, collapsible groups, open in browser
- Disk cache, instant start, background refresh (`refresh_interval`)
- Configurable key-sequence commands (PR data passed as `$PR_*` env vars, never interpolated)
- Background agents: `cc`/`cf`/`ca`/`cr` run headless Claude in the PR's worktree, detached (they outlive gh-prs), and push when done; one per PR, no global cap
- Agent column with a live spinner; a fixed-size agent box in the detail pane (recent actions while running, then the agent's DONE/changes/PUSHED/NEEDS YOU summary, which the prompt asks for); `ct` takes over the session in your opener (`claude --resume`)
- `D` merge + deploy prod: a script checks the deploy window, approval, checks, base branch and title scope (against the Deploy Services list and changed files), merges, runs Deploy Services per service, watches the runs, then Claude checks Datadog at +5/+10/+15/+20m; failures offer `V` (GitHub's revert PR). `DEPLOY_DRY_RUN=1` plans without merging
- Time limits for background jobs (`timeout`, default 45m, `D` 3h): a watchdog in the job's process group stops it and everything it started, even after gh-prs quits, and shows "Stopped: hit the time limit"
- Pluggable openers: global or per-command `opener` (any script) decides where commands open (Ghostty, tmux, herdr, …); `run` decides which agent
- Worktree reuse or creation (`<repo>-<branch-name>`), mark ready, merge (y/N), add label, copy URL/branch
- Organizing (`p…`): pin, snooze until it changes, move to a project, hide, all local in `organize.yml`; `pc` asks Claude (no tools) to reorganize, limited to grouping, previewed as a diff before anything is written
- `?` key list
- Awareness: ●/○ whose turn (with the reason in the detail pane, and `turn` as a `group_by`), ◆ changed since you last looked (and what changed), failing check names with `f` to load the error lines from the Actions log, Linear issue links (`i`), `/` filter

## Roadmap

| Phase | Theme | Scope | Status |
|---|---|---|---|
| 1 | Core | Fetch, nested grouping, navigation, detail pane, table, stacking, cache | ✅ Done |
| 2 | Actions | Configurable key→command bindings; `cc` opens Claude with prompt templates (fix CI, address comments, rebase, write description); checkout into a worktree; mark ready, merge, label, copy | ✅ Done |
| 2.5 | Background agents | Headless agent jobs, agent column, live activity, take over with `ct` | ✅ Done |
| 3 | Awareness | Whose turn it is ("on me" vs "on them"), changes since last viewed, failing check name and logs, Linear link, `/` filter | ✅ Done |
| 4 | Customization | Pins and overrides kept between sessions, Claude editing the grouping rules with a preview of the change | ✅ Done |
| 5 | Team | Agentic onboarding (Claude Code plugin: "set up gh-prs for me"), publish as a `gh` extension, neutral defaults | In progress |

## Open questions

- Should a stacked child always render under its parent, even when their statuses differ?
- PRs from forks: worktrees only work for branches pushed to the main repo.
- Human escalation for background agents: detect when an agent is blocked (a denied tool, a question) and surface it as "needs you", with `ct` as the way in.
- Deploy health checks cost about $1.20 each ($5 per deploy): fewer checks or a cheaper model?
- Switch background agents to `--permission-mode dontAsk`: verified that it makes the allowlist a hard limit.
- Should repos that don't require reviews count an approval as "Ready to merge"?
- Team rollout: one shared default config, or per-team presets?

## Changelog

- **2026-11-11:** Phase 4 shipped: pins, snoozes, local projects, hidden PRs, and Claude reorganizing with a diff preview.
- **2026-11-11:** Phase 3 shipped: whose turn, what changed, failing check logs, Linear links, filter.
- **2026-11-11:** Agent box: bounded summary of what an agent changed.
- **2026-11-11:** Time limits for background jobs.
- **2026-11-11:** Dropped `cd` (describe): it rewrote PR descriptions, and PRs are already written with one.
- **2026-11-11:** `D` merge + deploy prod with Datadog health checks, `V` revert PR.
- **2026-11-11:** Background agents: headless Claude jobs with an agent column, push on completion, `ct` to take over.
- **2026-11-12:** Agents push only to your own PRs; on others' they commit and leave the push to you. Agents refuse dirty worktrees, run with `dontAsk`, and use Sonnet.
- **2026-11-11:** Phase 2 shipped: commands, Claude launchers, worktrees. `cc` opens a new terminal window, configurable via `terminal`.
- **2026-11-11:** Phase 1 shipped: grouping, table, stacking, cache and background refresh.
