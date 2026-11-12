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

## 2. Audit: `make check`

Runs before every push once you've run `make hooks` in a clone (skip one push with `--no-verify`).

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

- Every merged PR is a release: `make release PR=<n>` (or `ci/workflows/release.yml` once
  enabled) tags it, patch by default or `minor`/`major` by label, and publishes the gh extension
  binaries for macOS and Linux. `RELEASE_DRY_RUN=1` rehearses.
- The macOS app is built on your Mac and attached: `make app-release TAG=vX.Y.Z`.
- Teammates update with `gh extension upgrade prs`.

## Keeping it healthy

Proposed, not set up (it would run an agent on a schedule, which costs Claude usage): a weekly
routine that runs `make check`, looks for dependency updates and new advisories, runs
`/ponytail-audit` and `/ponytail-debt`, and opens a PR with what it finds.
