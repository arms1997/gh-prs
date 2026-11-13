# Changelog

What changed in each release, for people who use gh-prs. Every PR that changes what a user sees or
does adds a line under **Unreleased** (`scripts/check-changelog.sh` checks this before a push).
`make release` turns that section into the release notes and renames it to the new version.

Each line goes under one of: **Added**, **Changed**, **Fixed**, **Removed**, **Security**.

## Unreleased

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
