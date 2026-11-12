# CI (drafted, not enabled)

GitHub only runs workflows under `.github/workflows/`. These live here until we've agreed on the
cost; enabling one is moving its file there.

| Workflow | Runs on | What it does |
|---|---|---|
| `check.yml` | every PR push, and every push to `main` | `make check`: gofmt, vet, `go test -race`, tsc, Vitest, `npm audit`, govulncheck |
| `release.yml` | each merged PR | next version from the PR's label (`major`, `minor`, else patch), tag, build the gh extension for macOS and Linux, publish a release |

The macOS app isn't built in CI: run `make app-release TAG=vX.Y.Z` on your Mac after a release.

## What it costs

This repo is private, so Actions minutes come out of your account's monthly allowance
(GitHub Free: 2,000; Pro: 3,000). Linux minutes count once, macOS minutes count 10 times. Every job
is rounded up to the whole minute. Release assets don't use Actions storage, and these workflows
upload no artifacts.

| Run | Runner | Estimate |
|---|---|---|
| check | Linux | ~5 min: setup 1, GTK/WebKit headers 1–2, npm ci and govulncheck 1, tests 1–2 |
| release | Linux | ~2 min |
| (not planned) the `.app` in CI | macOS | ~4 min, counted as ~40 |

**A busy month** (40 PRs, ~3 pushes each): 120 PR checks + 40 checks on `main` + 40 releases
≈ 120×5 + 40×5 + 40×2 = **~880 minutes**, under half of the Free allowance. The same month with the
app built in CI would add ~1,600, more than the whole allowance.

**Ways to spend less**, if we need them:

- Drop the check on pushes to `main`: the PR's check already ran on the merged code's parent. Saves ~200.
- Skip runs for PRs that only touch docs (`paths-ignore: ['**.md']`).
- Skip the GTK/WebKit headers by not compiling the desktop package on Linux. Saves ~1.5 min a run, but the desktop Go code is then only checked on your Mac.

**Before enabling**, check Settings → Billing and plans → Budgets: with a $0 budget for Actions,
runs stop at the allowance instead of being charged.

## Not verified yet

These haven't run anywhere. Expect a fix or two on the first run:
- the `webkit2_41` build tag for Wails on Ubuntu 24.04
- the release asset names `gh` expects (`gh-prs_<tag>_<os>-<arch>`)
