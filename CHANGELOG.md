# Changelog

What changed in each release, for people who use gh-prs. Every PR that changes what a user sees or
does adds a line under **Unreleased** (`scripts/check-changelog.sh` checks this before a push).
`make release` turns that section into the release notes and renames it to the new version.

Each line goes under one of: **Added**, **Changed**, **Fixed**, **Removed**, **Security**.

## Unreleased

### Changed
- Desktop app: a new icon, a white merge symbol on purple, instead of Wails's default.

## v0.4.0 — 2026-11-17

### Changed
- Desktop app: the PR list fills the window, with columns for whose turn it is and why, and each PR's
  agent. `e` (or a double-click) opens a PR on the whole page: `j`/`k` step to the next or previous PR,
  `[` `]` switch tabs, and `esc` goes back to the list where you left it. The details pane is gone,
  and in `config.yml`'s `columns:` the `next` and `detail` widths are dropped the next time you resize a column.
- Desktop app: a PR's page puts its title, next step and whose turn it is up top, with what's left
  before merge, its checklist and description on the left, and its agent, checks and details on the
  right. `1`–`9` pick the agent's suggested answers.
- Desktop app: a PR's page says what changed since you last looked (new commits, comments, CI,
  reviews, conflicts); leaving the page marks it seen.
- Desktop app: `space` in the list peeks at a PR: its header, agent and open question (`1`–`9` answer it),
  with `e` for the whole page and `space` or `esc` to close.

### Fixed
- Desktop app: commands that ask first (`M` merge, `D` merge + deploy prod, `V` revert) run when you
  answer Yes. Before, the question had only an OK button and the command was always cancelled. No is
  the default, so Return cancels.

## v0.3.3 — 2026-11-16

### Changed
- Desktop app: styles can now use Tailwind utilities built on the app's design tokens, the first step
  of moving off per-component CSS files. Nothing looks different yet.
- Desktop app: the shared buttons, badges, keys, tabs and dialogs use Tailwind. In the keys help,
  `pm` and `pa` show as two keys, like the commands listed under them.
- Desktop app: the PR list's status pieces and the Claude sessions page use Tailwind, with no visible change.
- Desktop app: the command palette, keys help, move and ask dialogs use Tailwind, with no visible change.
- Desktop app: the details pane and its tabs use Tailwind, with no visible change.
- Desktop app: the daily brief and Brief tab use Tailwind, and rendered markdown shares one stylesheet, with no visible change.
- Desktop app: the main page (top bar, quick filters, PR list, footer) uses Tailwind, with no visible change.
- Desktop app: the last shared helper classes are Tailwind utilities, which finishes the move to Tailwind.

## v0.3.2 — 2026-11-13

### Fixed
- Building the desktop app works again (v0.3.1's change of working folder broke binding generation);
  the app still moves to your home folder when opened from Finder.

## v0.3.1 — 2026-11-13

### Fixed
- A key sequence that isn't a command (like `cd`, which was removed) says "no command cd" instead of
  doing nothing, in both front ends.
- Desktop app: commands that open in your terminal (`M` merge, `L` label, `cc`…) say they opened there
  instead of showing ✓ before they've run, and no longer refresh before they finish. Without a
  worktree they open in your home folder, not `/`.

## v0.3.0 — 2026-11-13

### Added
- Desktop app: a daily brief (`b`, or the "Daily brief" button) lists what's worth doing first across
  your PRs: Claude's open questions, reviews people are waiting on, PRs ready to merge, and stuck ones.
- Desktop app: a Brief tab where Claude writes what's left on a PR before it can merge: reviews, open
  threads, failing checks and conflicts (`B`; about 10 seconds and a few cents per brief).

### Changed
- Claude calls that write text (the PR brief, reorganizing) no longer load your own Claude Code hooks
  and plugins, so they can't change the answer.

## v0.2.1 — 2026-11-13

### Fixed
- `cx`, `cy`, `ct` and `ci` find an agent started from the other front end or another gh-prs window,
  instead of saying no agent is running. Both front ends also show those agents in the list.

## v0.2.0 — 2026-11-13

### Added
- Desktop app: resize the list's columns and the details pane by dragging their edges (double-click
  resets, ←/→ when focused), or set them in `config.yml` under `columns:`. A wide details pane covers
  the list instead of squeezing it.

### Fixed
- Desktop app: long paths, branch names and URLs in the details pane wrap instead of running past
  their box.

## v0.1.2 — 2026-11-13

### Added
- `CHANGELOG.md`: what changed in each release, also used as the GitHub release notes.

## v0.1.1 — 2026-11-12

### Added
- Install instructions for teammates: the gh extension from a release, and the desktop app download.

### Changed
- The desktop app download is a universal build, for Apple silicon and Intel Macs.

## v0.1.0 — 2026-11-12

The first release. gh-prs had been in use from source before it; this lists what it shipped with.

### Added
- Terminal dashboard of your open PRs and review requests, grouped by project, status, repo or whose
  turn it is, refreshed in the background within GitHub's API budget.
- Commands on a PR by key: open it, check it out in a worktree, open a terminal, merge, and agents
  that run Claude in the background (`cc` get it mergeable, `cm` manage it until it merges).
- Conversations with agents: an agent's question shows as "needs you" with suggested answers; reply
  by resuming its session (`cy`), ask in a new one (`ci`), stop it (`cx`).
- Whose turn it is, what changed since you looked, failing check logs (`f`), Linear links, filter.
- Pins, snoozes, local projects and hidden PRs, and Claude reorganizing your groups with a diff preview.
- A macOS desktop app on the same core: grouped list with quick filters and a next step per PR,
  details with overview, Claude, checks and files tabs, a sessions page, and a ⌘K palette.
- `D`: merge, deploy to production through the Deploy Services workflow, and health-check the deploy.
- Agentic onboarding: a Claude Code plugin that interviews you and writes your config.

### Security
- Agents push only to your own PRs; on other people's PRs a push-URL rewrite blocks pushes.
- PR text written by other people is sanitized and passed to commands as environment variables,
  never interpolated into shell commands or prompts.
- Built with Go 1.26.11 for two standard library advisories (GO-2026-6809, GO-2026-6859).

### Fixed
- Stopping an agent with a time limit no longer leaves its watchdog process running.
