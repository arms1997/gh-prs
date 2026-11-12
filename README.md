# gh-prs

A terminal dashboard for your GitHub pull requests. It groups them by project and by what needs doing, shows CI, review and stacking status at a glance, and can hand any PR to a background Claude agent while you keep working.

```
 pr                                    lines      ci      review       unres  merge      agent
▾ My PRs (21)
  ▾ tool-display-name (6)
    ▾ Needs action (3)
      #20465 fix(agent-registry): keep… +23 −36     ✗ fail  waiting      1                 ⠹ fix CI 4m
      ↳ #20261 (on #20227) feat(agen… +126 −106   ✗ fail  waiting      2                 ✓ fix CI
      #12965 fix: update load search…  +4 −2       ✓ pass  ✓ approved   1      conflict
    ▾ Waiting for review (2)
      #20326 feat(agent-registry): ad… +9790 −0    ✓ pass  waiting
      ↳ #20227 feat(agent-registry): … +1413 −134  ✓ pass  waiting      1
▾ Review requests (13)
  ▸ GoAugment/augment-services (6)
```

The detail pane on the right shows the selected PR's description, branches, stats, and a box with what its agent did.

## Install

### With Claude (recommended)

In Claude Code:

```
/plugin install gh-prs --marketplace arms1997/gh-prs
```

(On Claude Code before 2.1.275: `/plugin marketplace add arms1997/gh-prs`, then `/plugin install gh-prs@gh-prs`.)

Then say **"set up gh-prs for me"**. Claude checks the prerequisites, clones and builds gh-prs, and interviews you: where your repos are cloned, how to group your PRs, which terminal commands open in, which commands and agents to keep, and your repos' conventions for agents to follow. It writes `~/.config/gh-prs/` and checks it with `gh prs check`. Say it again any time to change your setup.

### By hand

Requires [`gh`](https://cli.github.com) (logged in) and Go 1.26+. Background agents need [`claude`](https://claude.com/claude-code). The Herdr opener and the deploy command also need `jq`.

```sh
git clone git@github.com:arms1997/gh-prs.git && cd gh-prs
go build -o gh-prs .
gh extension install .     # links this directory; rebuild to update
gh prs
```

On first run, gh-prs writes a commented default config to `~/.config/gh-prs/config.yml`.

## Using it

| Keys | |
|---|---|
| `j` `k` `↑` `↓` | move |
| `gg` `G`, `ctrl+d` `ctrl+u` | top / bottom, half page |
| `enter` `h` `l` | fold, collapse, expand a group |
| `e` | focus the detail pane to scroll it (`esc` back) |
| `/` | filter by title, number, repo, branch, author or Linear ID (`enter` keeps it, `esc` clears) |
| `f` | load the failing check's log, around its errors |
| `o` | open the PR in the browser |
| `i` | open the PR's Linear issue |
| `r` | refresh now (it also refreshes every 2 minutes in the background) |
| `p…` | organize: `pp` pin · `ps` snooze · `pm` move to project · `ph` hide · `pa` show hidden · `pc` ask Claude |
| `?` | every key, including your commands |
| `q` | quit |

**Groups.** Each section is a GitHub search. Within a section, PRs nest by `project` (labels starting with `proj:`), `status`, or `repo`, in whatever order you configure.

**Status buckets.**
- **Draft:** the PR is a draft.
- **Needs action:** changes requested, failing CI, or a merge conflict.
- **Ready to merge:** approved, with CI green or no checks.
- **Waiting for review:** everything else.

**Columns.**
- `lines`: lines added and removed.
- `ci`, `review`: CI and review state.
- `unres`: unresolved review threads.
- `merge`: merge conflicts.
- `agent`: the PR's background agent.

The reasons a PR needs action show as solid colored badges.

**Whose turn.** Each row starts with ● (your move) or ○ (theirs).
- **Your PRs are your turn when:** they're drafts, need action, have a thread where someone else spoke last, or are ready to merge.
- **Review requests are your turn** until you've reviewed them since their latest commit.

The detail pane says why. `turn` also works as a `group_by` key.

**What changed.** ◆ marks a PR that changed since you last selected it. Selecting it lists what changed in the detail pane (new commits, new comments, CI, review state, conflicts) and clears the mark.

**Failing checks and Linear.** The detail pane names each failing check; `f` loads the failure from the GitHub Actions log. Issue IDs like `AGT-915` are found in the title or branch and link to `linear_workspace`.

**Organizing.** Your own arrangement is kept in `~/.config/gh-prs/organize.yml`, keyed by `owner/repo#number`. Nothing in it goes to GitHub, and it works on anyone's PRs.
- `pp` **pins** a PR to the top of its group (★).
- `ps` **snoozes** it until something happens on it: a commit, comment, CI change or review. Then it comes back marked ◆.
- `pm` **moves** it to a project you type, overriding its `proj:` label. Leave the name empty to undo.
- `ph` **hides** it for good. `pa` shows hidden (⊘) and snoozed (z) PRs, so you can undo either with the same key.

**Asking Claude to reorganize (`pc`).** Type what you want, e.g. "put the tool display name PRs in their own project and group My PRs by turn". Claude proposes new versions of your config and organize file, and the detail pane shows the diff: `y` applies (the old config is kept as `config.yml.bak`), `n` discards.
- **Text only:** Claude runs with no tools, so it can only propose text.
- **Grouping only:** a proposal may change sections, project labels, pins, projects and hidden PRs. Anything that touches commands, the opener, `repos_dir` or the prompts is refused, because PR titles in its input are written by other people.
- **Validated:** a proposal that doesn't load is refused too. A request costs about $0.10.

**Stacking.** A PR whose base branch is another of your PRs' head branch is indented under it with `↳`. If the parent sits in a different group, it's marked `↳ (on #N)`.

**Cache.** gh-prs starts instantly from `~/.cache/gh-prs/prs.json`, then refreshes in the background without moving your cursor.

## Commands

Commands are key sequences defined in the config. The defaults:

| Keys | Command | What it does |
|---|---|---|
| `cc` | get it mergeable | background agent: fix checks, address comments, resolve conflicts |
| `cf` | fix CI | background agent: find and fix failing checks |
| `ca` | address comments | background agent: address unresolved review threads |
| `cr` | rebase | background agent: rebase onto the base branch, push with `--force-with-lease` |
| `cm` | manage | background agent on `/loop`: every 15m resolves conflicts, rebases when behind and addresses new review comments, until the PR merges (◷ between rounds; stops after 8h or $10) |
| `ct` | take over agent | open the PR's last agent session in your opener (`claude --resume`) |
| `cx` | stop agent | stop the PR's running agent, a `cm` loop included, and everything it started (asks first) |
| `w` | shell in worktree | open a shell in the PR's worktree |
| `D` | merge + deploy prod | merge, deploy, then health-check prod (see below) |
| `V` | open revert PR | open a revert PR for a merged PR |
| `R` `M` `L` | mark ready, merge, add label | |
| `y` `Y` | copy URL, copy branch | |

Commands marked y/N in the config ask before running (`D`, `V`, `M`).

### Background agents

`cc`, `cf`, `ca` and `cr` run `claude -p` headless in the PR's worktree:
- **Detached from gh-prs:** they keep running if you quit.
- **Live column:** the `agent` column shows a spinner while they run.
- **Agent box:** the detail pane's box shows their recent actions, then a summary of what they changed, the commit they pushed, and anything left for you.
- **Logs:** every run is logged in `~/.cache/gh-prs/jobs/`.

What to know before using them:
- **They push to your PRs only.** When done, an agent commits and pushes to the PR's head branch. On someone else's PR (a review request) it commits but doesn't push: `git push` is blocked for that job, and the agent box says `PUSHED: NO`. Review the commit in the worktree and push it yourself.
- **Allowed tools:** `--allowedTools` pre-approves file edits, `git`, `gh` and common package managers and test runners. Anything else follows your own Claude Code permission mode and allow rules (`defaultMode` in `~/.claude/settings.json`): in `auto` mode Claude Code decides, in `default` mode a headless run refuses it.
- **Clean worktrees only:** an agent won't start in a worktree with uncommitted or untracked files, since it commits and pushes when done.
- **Your Claude login:** they run as `env -u ANTHROPIC_API_KEY claude …`, so they use your claude.ai login even if an API key is set in your environment.
- **Sonnet:** they run with `--model sonnet`, as do `pc` and the deploy health checks. Override a command in your config to use another model.
- **One per PR:** only one agent runs on a PR at a time, since they'd share its worktree. There's no global limit.
- **Time limit:** each job stops after 45 minutes (set `timeout` per command, `"0"` for none). A watchdog inside the job stops it and everything it started, even if gh-prs isn't running.

### Deploy and monitor (`D`)

`D` is written for repos with a `Deploy Services` workflow (`.github/workflows/dispatch_deploy.yml` taking `deploy-service` and `deploy-env`). It runs `scripts/deploy-monitor.sh` as a background job.

```
check, before anything changes:
  prod deploy window (Mon–Thu 3AM–7PM PT, Fri until 2PM ET)
  PR open, approved, checks clean, based on the default branch
  service(s) from the title scope, e.g. fix(agent-registry), checked against the
  workflow's service list and the PR's changed files
merge ─► run Deploy Services per service with env=prod ─► watch the runs
─► Claude checks Datadog at +5, +10, +15, +20 min (read-only tools)
─► ✓ healthy  or  ✗ unhealthy, with "press V to open a revert PR"
```

If any check fails, nothing is merged. Repos without the workflow are merged only. The script reads these environment variables:
- `DEPLOY_DRY_RUN=1`: run the checks and print the plan only.
- `DEPLOY_CHECK_MINUTES="10 20"`: when to run health checks, in minutes after the deploy.
- `DATADOG_MCP`: the name of your Datadog MCP server.
- `DEPLOY_RUNNER`: which GitHub Actions runner the deploy uses.

Each health check costs about $1.20.

## Desktop app (preview)

`desktop/` is the same gh-prs as a macOS app ([Wails](https://wails.io): a web UI on the same Go code). It reads the same config, cache, organize.yml and agent jobs, so you can use it alongside the terminal UI.

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@latest   # once; needs Node.js too
cd desktop && wails build                                    # → desktop/build/bin/gh-prs.app
open build/bin/gh-prs.app
```

It has the grouped list, the detail pane with the description rendered as Markdown, the agent box, filtering, and the same keys (`j`/`k`, `o`, `r`, `/`, `p…`, command keys), plus clickable command buttons. Commands that need a terminal (`interactive: true`) open in your opener instead. Not there yet: `pc`, failing-check logs (`f`), "changed since you looked", and notifications.

The app is self-signed, so a teammate opening a copy you built gets a macOS warning; build it yourself.

## Configuration

`~/.config/gh-prs/config.yml` is layered over the built-in defaults in [`defaults/`](defaults): any key you set replaces the default, and unknown keys are an error. The generated file documents every option. The main ones:

```yaml
project_label_prefix: "proj:"
refresh_interval: 2m            # 0 turns background refresh off
repos_dir: ~/augment            # owner/repo is expected at <repos_dir>/repo
opener: open -na Ghostty.app --args -e
linear_workspace: go-augment    # Linear links: linear.app/<workspace>/issue/AGT-915
prompt_context_file: context.md # prepended to every agent prompt

sections:
  - title: My PRs
    query: is:pr is:open author:@me archived:false sort:updated-desc
    group_by: [project, status]
  - title: Review requests
    query: is:pr is:open user-review-requested:@me archived:false sort:updated-desc
    group_by: [repo]
```

### Commands

The built-in commands live in [`defaults/commands/`](defaults/commands), grouped as `agents.yml`, `github.yml` and `deploy.yml`. Setting `commands:` in your config replaces all of them.

```yaml
commands:
  - key: cf
    name: fix CI
    worktree: true      # run in the PR branch's worktree, creating it if needed
    background: true    # detached agent job; stdout must be claude stream-json
    timeout: 45m
    run: env -u ANTHROPIC_API_KEY claude -p "$(cat "$PR_PROMPT_FILE")" --session-id "$PR_SESSION_ID" ...
    prompt_file: fix-ci.md   # or prompt: with the text inline
```

Command options:

| Option | Meaning |
|---|---|
| `prompt_file`, `prompt` | the agent's task, from a file or inline (a Go template, after the prompt context) |
| `worktree` | run in the PR branch's worktree, creating it if needed |
| `background` | run as a detached agent job |
| `terminal` | hand the command to the opener |
| `interactive` | take over this terminal |
| `opener` | override the global opener for this command |
| `agent_session` | needs the PR's last agent job |
| `timeout` | background only: stop the job after this long |
| `confirm` | ask y/N first |
| `refresh` | reload PRs after it succeeds |

Commands receive PR data only as environment variables. Use `"$PR_URL"` in `run`, never a template.

| Variable | When |
|---|---|
| `PR_NUMBER` `PR_REPO` `PR_URL` `PR_HEAD` `PR_BASE` `PR_KEY` `PR_ACTION` `GH_PRS_DIR` | always |
| `PR_WORKTREE` | `worktree: true` |
| `PR_PROMPT_FILE` | `prompt` is set |
| `PR_SESSION_ID` | `background: true` (pass to `claude --session-id`) |
| `PR_AGENT_SESSION` | `agent_session: true` |
| `PR_LAUNCH` | `terminal: true` (opener only) |

### Prompts

Agent prompts are Markdown files in [`defaults/prompts/`](defaults/prompts): `context.md`, which every agent gets first, then one per command (`fix-ci.md`, `address-comments.md`, …). A prompt name is looked up in `~/.config/gh-prs/prompts/` before the built-ins, so you can replace any prompt by adding a file with its name. Absolute and `~/` paths work too.

Prompts can use `{{.Number}} {{.Repo}} {{.URL}} {{.Head}} {{.Base}}`. The PR title is left out on purpose, so a PR author's text never becomes agent instructions.

### Openers

The opener decides where `terminal: true` commands (like `ct` and `w`) open. It's called as `<opener> "$1"`, where `$1` (also `$PR_LAUNCH`) is a launcher script. The script `cd`s into the worktree, runs the command, and leaves a shell open. The opener also gets every `$PR_*` variable.

```yaml
opener: open -na Ghostty.app --args -e          # a new Ghostty window
opener: tmux new-window -n "pr-$PR_NUMBER"      # a tmux window
opener: /path/to/gh-prs/openers/herdr.sh        # a Herdr space per PR, a tab per command
```

## Security

- **No shell injection:** PR data reaches commands only through environment variables. Launcher scripts single-quote every value. A test pushes a branch name full of `$(…)`, backticks and quotes through every path.
- **Terminal escapes stripped:** control characters are removed from PR titles, bodies, branch names and agent output before rendering.
- **Private files:** the cache, job logs and prompt files are readable only by you.

## Development

```sh
go test ./...        # unit tests, plus end-to-end tests of jobs, worktrees and launchers
go vet ./...
```

The terminal UI is package `main` in the root; everything it shows and does lives in package `core/`, which has no UI code, so another front end can reuse it.

| Path | What's in it |
|---|---|
| `main.go`, `ui.go` | entry point; the Bubble Tea model, Update and View |
| `keys.go`, `table.go`, `detail.go`, `styles.go` | keyboard handling, the PR list, the detail pane and agent box, colors |
| `checks.go`, `reorganize.go` | UI wrappers for failing-check logs and `pc` |
| `core/github.go`, `core/group.go`, `core/cache.go` | fetching PRs and the rate limit, grouping, whose turn, filtering and stacking, the disk cache |
| `core/seen.go`, `core/checks.go` | what changed since you last looked; failing check logs |
| `core/organize.go`, `core/reorganize.go` | pins, snoozes, projects and hidden PRs; asking Claude to reorganize |
| `core/config.go`, `core/commands.go` | config layering and prompt lookup; validating and preparing commands |
| `core/launcher.go`, `core/worktree.go` | launcher scripts, shell quoting and time limits; worktrees |
| `core/jobs.go`, `core/stream.go` | background agent jobs; parsing Claude's stream-json |
| `defaults/` | built-in config, commands and prompts, embedded in the binary |
| `openers/`, `scripts/` | the Herdr opener; the deploy-and-monitor script |

`VISION.md` holds the product vision, roadmap and open questions.

## Limitations

- **No pagination:** each section shows up to 100 PRs.
- **No forks:** worktrees need the branch in the main repo.
- **macOS-only defaults:** `y`/`Y` use `pbcopy`, and the default opener uses Ghostty on macOS.
- **Augment conventions:** `D` follows Augment's deploy workflow and safe-hours rules.
