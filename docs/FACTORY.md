# How gh-prs is built

Every change goes through the same four stages. Each stage has a gate that is a command, so a
person and an agent follow the same process.

```
IMPLEMENT ─────────► AUDIT ─────────────► REVIEW ──────────────► DEPLOY
branch + PR          make check            PR checks             merge
CLAUDE.md            (pre-push hook)       /code-review          make release PR=<n>
                     tests at 4 layers     /security-review      make app-release TAG=<v>
                     vulnerability scans   you approve           VISION changelog
```

## 1. Implement

- One branch and PR per change, from `origin/main` or stacked on the PR it builds on.
- `CLAUDE.md` is the brief for any agent: layout, rules, definition of done.
- New behaviour comes with its test, at the lowest layer that can show it works (below).
- A change users would notice adds a line under `## Unreleased` in `CHANGELOG.md`.
  `scripts/check-changelog.sh` refuses a push that changes the app, terminal UI, core, defaults,
  plugin or openers without one; tests and docs don't need one.

## 2. Audit: `make check`

Runs before every push, with the changelog check, once you've run `make hooks` in a clone (skip one
push with `--no-verify`).

| Check | Catches |
|---|---|
| `gofmt`, `go vet` | formatting, suspicious Go |
| `go test -race ./...` | Go unit tests, data races, and the end-to-end agent flows (`core/e2e_test.go`: real processes, worktrees and git against a fake `claude`) |
| `tsc`, `npm test` | front-end types, Vitest unit tests (list, next step, before-merge, sanitizing) |
| `npm audit`, `govulncheck` | known vulnerabilities in npm packages, Go modules and the Go standard library |

Test layers, cheapest first:

1. **Unit:** Go next to the code, Vitest next to `desktop/frontend/src/lib/`.
2. **End-to-end agent flows:** `core/e2e_test.go`.
3. **UI end to end** (not built yet): Playwright against `wails dev` with fixture data, and
   `teatest` snapshots of the terminal UI. Needs a `GH_PRS_FIXTURE` hook in `core.FetchAll`.
4. **Contract smoke** (not built yet): a script you run by hand that checks the real `claude` and
   `gh` still behave as assumed (`--resume` keeps the session id, stream-json shape). Costs cents.

## 3. Review

- The PR template asks for Problem, Changes, Testing.
- Before asking for review, run `/code-review` and `/security-review` in Claude Code on the branch
  and fix what they find.
- You approve and merge. CI would re-run `make check` here once enabled (`ci/`, with costs).

## 4. Deploy

- Every merged PR is a release, made right after it merges: `make release PR=<n>` (or
  `ci/workflows/release.yml` once enabled) picks the version (patch by default, `minor`/`major` by
  label), uses CHANGELOG.md's Unreleased section as the release notes, commits "chore: release
  vX.Y.Z" to `main` renaming that section to the version, tags that commit, and publishes the gh
  extension binaries for macOS and Linux. `RELEASE_DRY_RUN=1` rehearses and shows the notes.
- The release commit is the one push to `main` outside a PR; it needs the ruleset's admin bypass.
- The macOS app is built on your Mac and attached: `make app-release TAG=vX.Y.Z`.
- Teammates update with `gh extension upgrade prs`.

## Keeping it healthy

Proposed, not set up (it would run an agent on a schedule, which costs Claude usage): a weekly
routine that runs `make check`, looks for dependency updates and new advisories, runs
`/ponytail-audit` and `/ponytail-debt`, and opens a PR with what it finds.
