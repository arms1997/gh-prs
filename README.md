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
| `o` | open the PR in the browser |
| `r` | refresh now (it also refreshes every 2 minutes in the background) |
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
| `ct` | take over agent | open the PR's last agent session in your opener (`claude --resume`) |
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
- **They push.** When done, an agent commits and pushes to the PR's head branch. That includes other people's PRs from your review requests.
- **Allowed tools:** they work within `--allowedTools` (file edits, `git`, `gh`, common package managers and test runners) plus your Claude settings' allow rules. If your settings use `defaultMode: auto`, other commands judged safe may also run. Switching the agents to `--permission-mode dontAsk` makes the list a hard limit; the deploy health checks already use it.
- **Your Claude login:** they run as `env -u ANTHROPIC_API_KEY claude …`, so they use your claude.ai login even if an API key is set in your environment.
- **One per PR:** only one agent runs on a PR at a time, since they'd share its worktree. There's no global limit.
- **Time limit:** each job stops after 45 minutes (set `timeout` per command, `"0"` for none). A watchdog inside the job stops it and everything it started, even if gh-prs isn't running.
- **Worktree changes:** an agent's commit can pick up uncommitted changes already in the worktree.

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

## Configuration

`~/.config/gh-prs/config.yml` is layered over the built-in defaults in [`defaults/`](defaults): any key you set replaces the default, and unknown keys are an error. The generated file documents every option. The main ones:

```yaml
project_label_prefix: "proj:"
refresh_interval: 2m            # 0 turns background refresh off
repos_dir: ~/augment            # owner/repo is expected at <repos_dir>/repo
opener: open -na Ghostty.app --args -e
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

| Path | What's in it |
|---|---|
| `main.go`, `ui.go` | entry point; the Bubble Tea model, Update and View |
| `keys.go`, `table.go`, `detail.go`, `styles.go` | keyboard handling, the PR list, the detail pane and agent box, colors |
| `github.go`, `group.go`, `cache.go` | fetching PRs, grouping and stacking, the disk cache |
| `config.go`, `commands.go` | config layering and prompt lookup; validating and preparing commands |
| `launcher.go`, `worktree.go` | launcher scripts, shell quoting and time limits; worktrees |
| `jobs.go`, `stream.go` | background agent jobs; parsing Claude's stream-json |
| `defaults/` | built-in config, commands and prompts, embedded in the binary |
| `openers/`, `scripts/` | the Herdr opener; the deploy-and-monitor script |

`VISION.md` holds the product vision, roadmap and open questions.

## Limitations

- **No pagination:** each section shows up to 100 PRs.
- **No forks:** worktrees need the branch in the main repo.
- **macOS-only defaults:** `y`/`Y` use `pbcopy`, and the default opener uses Ghostty on macOS.
- **Augment conventions:** `D` follows Augment's deploy workflow and safe-hours rules.
