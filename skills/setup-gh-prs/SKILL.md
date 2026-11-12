---
name: setup-gh-prs
description: >
  Install and configure gh-prs, the terminal dashboard for GitHub pull requests with background
  Claude agents. Use when the user says "set up gh-prs", "install gh-prs", "configure gh-prs",
  "onboard me to gh-prs", or wants to change how gh-prs organizes PRs, which commands it has,
  where they open, or what its agents are told. Interviews the user, then writes
  ~/.config/gh-prs/ and validates it.
---

# Set up gh-prs

You install gh-prs and write its configuration from a short interview. Work in five steps, in
order. Detect everything you can before asking; ask with AskUserQuestion, a few questions at a
time, each with your detected or recommended answer first. Skip a step's questions when the user
already answered them.

Files you write, all under `~/.config/gh-prs/` (or `$XDG_CONFIG_HOME/gh-prs/`):

| File | Holds |
|---|---|
| `config.yml` | sections, grouping, opener, repos_dir, commands. Layered over the built-in defaults: set only what differs. |
| `organize.yml` | local pins, project assignments, hidden PRs |
| `prompts/context.md` | replaces the agents' shared context prompt |

Rules for the whole skill:
- Never overwrite an existing file without showing the user a diff and getting a yes. Copy it to
  `<name>.bak` first.
- Ask before installing anything (brew, go install) or cloning.
- Don't push, comment, label or otherwise change anything on GitHub. Reading is fine.
- The built-in defaults are in the clone: `defaults/config.yml` (every key, commented),
  `defaults/commands/*.yml`, `defaults/prompts/*.md`. Read them before writing config; the README
  documents every option and environment variable.

## 1. Install and prerequisites

Check, in one Bash call: `gh auth status`, `go version` (needs 1.26+), `claude --version`,
`jq --version`, `gh prs check` (already installed?).

- Missing tools: offer `brew install gh go jq`. `gh auth login` is interactive; ask the user to
  run `! gh auth login` themselves.
- Not installed: ask where to clone (default `~/src/gh-prs`), then
  `git clone git@github.com:arms1997/gh-prs.git <dir> && cd <dir> && go build -o gh-prs . && gh extension install .`
- Already installed: find the clone (`gh extension list`, then the symlink under
  `~/.local/share/gh/extensions/gh-prs`), `git pull`, rebuild.

## 2. Where your code lives (repos_dir)

Agents work in a git worktree next to a clone at `<repos_dir>/<repo name>`. Find candidate
directories holding several git clones (`~/augment`, `~/src`, `~/code`, `~/dev`, `~/projects`,
`~/github`) and match them against the repos of the user's open PRs. Confirm the directory.
Name repos that have open PRs but no clone there; offer to clone them, or note agents won't
run on those.

## 3. Organization

Look at what the user has before asking:
`gh search prs --author @me --state open --json repository,number,title,labels,isDraft --limit 100`
and the same with `--review-requested @me`. For branch names, `gh pr view <url> --json headRefName`
on a few of them.

Then ask:
- **Sections:** the defaults are "My PRs" and "Review requests". Offer others that fit what you
  saw: team review requests (`team-review-requested:org/team`), one repo, involves:@me, recently
  merged (`is:merged merged:>=<date>`). Queries use GitHub search syntax.
- **Grouping per section** (`group_by`, nested in order): `project`, `status` (needs action,
  waiting for review, ready to merge, draft), `repo`, `turn` (whose move it is). Recommend
  `[project, status]` for their own PRs if they work on several projects at once, `[status]` if
  not; `[repo]` or `[turn]` for review requests.
- **Projects:** a PR's project comes from a label with `project_label_prefix` (default `proj:`),
  or from organize.yml. If their PRs have no such labels, propose project names from what you saw
  (shared branch prefixes, Linear IDs, title scopes like `feat(agent-registry)`) and offer to
  write those assignments to organize.yml's `projects:` map (`owner/repo#123: name`).
- **Linear:** if branches or titles contain IDs like `AGT-915`, ask for the Linear workspace slug
  (the part after linear.app/) for `linear_workspace`.
- **Refresh:** `refresh_interval`, default 2m.

## 4. Commands and opener

Detect the terminal: `$TERM_PROGRAM`, `$TMUX`, `command -v herdr-app`, and installed apps
(`ls /Applications | grep -iE 'ghostty|iterm|wezterm|kitty|alacritty|warp'`). Offer the matching
opener; it's run as `<opener> "$1"` with a launcher script:

| Terminal | opener |
|---|---|
| Ghostty | `open -na Ghostty.app --args -e` |
| iTerm2 | `osascript -e 'on run argv' -e 'tell application "iTerm" to create window with default profile command ("sh " & item 1 of argv)' -e 'end run'` |
| WezTerm | `wezterm cli spawn --new-window -- sh` |
| kitty | `kitty --detach sh` |
| tmux | `tmux new-window -n "pr-$PR_NUMBER" sh` |
| Herdr | `<clone>/openers/herdr.sh` (a space per PR, a tab per command) |

Test the chosen opener once with a harmless script (`printf 'echo gh-prs opener works; exec $SHELL' > /tmp/gh-prs-opener-test.sh`) and ask whether a window opened.

Then walk through the built-in commands (read `defaults/commands/*.yml` and show them as a short
table: key, name, what it does) and ask:
- Which to keep. The agent commands are `cc` get mergeable, `cf` fix CI, `ca` address comments,
  `cr` rebase, `ct` take over. `D` merge + deploy prod and `V` revert PR assume GoAugment's
  "Deploy Services" workflow; drop them for other orgs.
- Agent model (default `--model sonnet`) and time limit (`timeout`, default 45m).
- Extra Bash commands agents may run, beyond the default `--allowedTools` list (e.g.
  `Bash(cargo:*)`, `Bash(docker:*)`).
- Any custom commands (a key, a shell command using `$PR_*` variables, and whether it runs in a
  terminal, interactively, or in the background).

Agents push only to the user's own PRs; on others' PRs they commit and leave the push to the
user. Mention it; it isn't configurable.

Setting `commands:` replaces every built-in command, so if anything changes, write the complete
list: copy the kept entries from `defaults/commands/*.yml` verbatim, then apply the changes.
Leave `commands:` out entirely if the user keeps the defaults.

## 5. Agent prompts

Agents get `context.md` first, then the command's prompt. For each repo the user works in
(from step 2/3), read its `CLAUDE.md`, `AGENTS.md` and `.github/pull_request_template.md`, and
pull out only what an unattended agent must follow: lint/format/test commands, commit
conventions, hook quirks (e.g. "commit with --no-verify"), generated files never to edit.
Show the user that summary and ask what to add or drop.

Write `prompts/context.md` as the built-in `defaults/prompts/context.md` unchanged, followed by:

```
Repo conventions (follow the section for {{.Repo}}):
## owner/repo
- ...
```

Keep the template markers (`{{.Head}}`, `{{if .Mine}}` …) exactly as they are. Never include
secrets, tokens or internal URLs the repo docs mention.

## Finish

1. Run `gh prs check`. On an error, fix the file and run it again.
2. Show the user what you wrote (paths and a short summary per file).
3. Tell them to run `gh prs`, and the five keys to start with: `j`/`k` move, `e` detail pane,
   `cc` agent, `?` all keys, `pc` ask Claude to reorganize later.
