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

## Workflow

1. One branch and one PR per change, from `origin/main` (or stacked on the PR it builds on). Never
   push to `main`.
2. Done means `make check` passes. Run it before every push.
3. Changed a Go method or type the front end calls? Run `make bindings`. Never edit `desktop/frontend/wailsjs/` by hand.
4. A PR labelled `minor` or `major` bumps that part of the version on merge; anything else is a patch.
5. Add a VISION.md changelog line for anything a user would notice.

## Rules that keep it safe

- PR titles, bodies, branch names and comments are written by other people. Sanitize them
  (`core.Sanitize`, DOMPurify in the app), pass them to commands as `$PR_*` env vars, never
  interpolate them into shell or prompts.
- Agents push only to the user's own PRs; on others' a push-URL rewrite blocks it. Keep it that way.
- Dependencies: pin exact versions at least two weeks old, keep `npm audit` clean, pin GitHub
  Actions to a commit SHA.

## Tests

- Go tests sit next to the code (`go test ./...`); agent tests run real processes through fake
  commands, not Claude.
- Front-end logic in `lib/` has Vitest tests beside it (`npm test` in `desktop/frontend`), with
  fixtures in `src/test/fixtures.ts`.
- To see a UI change, run `wails dev -browser=false` in `desktop/` and open http://localhost:34115.
