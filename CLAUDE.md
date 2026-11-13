# gh-prs

A PR dashboard with two front ends on one core. VISION.md says what it is and where it's going;
README.md says how to use it.

## Layout

- `core/`: all behaviour (GitHub fetch, grouping, commands, agent jobs, organize). No UI code.
- `*.go` at the root: the terminal UI (Bubble Tea), package main.
- `desktop/`: the macOS app (Wails). `app.go` exposes core to the front end.
- `desktop/frontend/src/`: `pages/` (whole screens) compose `features/` (detail, claude, palette,
  organize, ask), which use `components/` (ui, pr). Logic without UI goes in `lib/`, shared state in
  `state/AppState.tsx`.
- `defaults/`: built-in config, commands and agent prompts, embedded in the binary.

## Styles (desktop app)

- Style with Tailwind utilities in `className`. No new CSS files or global class names.
- Colors, radii, shadows and type sizes come from the app's tokens (`styles/tailwind.css`): `bg-muted`,
  `text-dim`, `text-xs`. Tailwind's default palette is off, and dark mode follows the tokens, so
  no `dark:` variants.
- Variants are literal class strings in a map (`{ok: 'text-ok-fg', bad: 'text-bad-fg'}`), never
  built from pieces: Tailwind only generates classes written out in full in the source.
- A pattern used in more than one place becomes a component in `components/ui` (Button, Badge,
  Input, Heading, BoxList), not a shared class.
- CSS lives only in `styles/`: tokens, base (reset, scrollbars), the Tailwind entry, and
  `prose.css` for markdown rendered as HTML, which has no class names to put utilities on.

## Workflow

1. One branch and one PR per change, from `origin/main` (or stacked on the PR it builds on). Never
   push to `main`; the only exception is the release commit `scripts/release.sh` makes.
2. Done means `make check` passes. `make hooks` runs it before every push. docs/FACTORY.md has the
   whole pipeline, from branch to release.
3. Changed a Go method or type the front end calls? Run `make bindings`. Never edit `desktop/frontend/wailsjs/` by hand.
4. A PR labelled `minor` or `major` bumps that part of the version on merge; anything else is a patch.
5. Anything a user would notice gets a line under `## Unreleased` in CHANGELOG.md (Added, Changed,
   Fixed, Removed, Security), written for the people who use gh-prs; the pre-push hook checks it.
   It becomes the release notes. Notable milestones also get a VISION.md changelog line.

## Rules that keep it safe

- PR titles, bodies, branch names and comments are written by other people. Sanitize them
  (`core.Sanitize`, DOMPurify in the app), pass them to commands as `$PR_*` env vars, never
  interpolate them into shell or prompts.
- Agents push only to the user's own PRs; on others' a push-URL rewrite blocks it. Keep it that way.
- Third-party dependencies: pin exact versions at least two weeks old, keep `npm audit` and
  `govulncheck` clean, pin GitHub Actions to a commit SHA. Go toolchain security releases are the
  exception: take them as they come (`toolchain` in go.mod).

## Tests

- Go tests sit next to the code (`go test ./...`); agent tests run real processes through fake
  commands, not Claude.
- `core/e2e_test.go` runs the built-in agent commands end to end (ask, manage, reply, stop)
  against `core/testdata/fake-claude`, which records how it was called. Change an agent flow,
  extend it.
- Front-end logic in `lib/` has Vitest tests beside it (`npm test` in `desktop/frontend`), with
  fixtures in `src/test/fixtures.ts`.
- To see a UI change, run `wails dev -browser=false` in `desktop/` and open http://localhost:34115.
