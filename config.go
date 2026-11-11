package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Section struct {
	Title   string   `yaml:"title"`
	Query   string   `yaml:"query"`
	GroupBy []string `yaml:"group_by"`
}

type Config struct {
	ProjectLabelPrefix string    `yaml:"project_label_prefix"`
	RefreshInterval    string    `yaml:"refresh_interval"`
	ReposDir           string    `yaml:"repos_dir"`
	Opener             string    `yaml:"opener"`
	PromptContext      string    `yaml:"prompt_context"`
	Sections           []Section `yaml:"sections"`
	Commands           []Command `yaml:"commands"`

	refresh time.Duration // parsed RefreshInterval; 0 disables auto-refresh
}

const defaultConfigYAML = `# gh-prs config. Anything left out falls back to these defaults.
#
# sections.query: GitHub search syntax (https://docs.github.com/en/search-github/searching-on-github/searching-issues-and-pull-requests)
# sections.group_by: any of project, status, repo; nested in the order given
# refresh_interval: background refresh, e.g. 2m; 0 turns it off
project_label_prefix: "proj:"
refresh_interval: 2m
sections:
  - title: My PRs
    query: is:pr is:open author:@me archived:false sort:updated-desc
    group_by: [project, status]
  - title: Review requests
    query: is:pr is:open user-review-requested:@me archived:false sort:updated-desc
    group_by: [repo]

# Where your clones live: owner/repo is expected at <repos_dir>/repo.
# New worktrees are created next to the clone as <repo>-<branch-name>.
repos_dir: ~/augment

# opener: how commands with terminal: true get opened (a new window, a multiplexer tab,
# your own script). It runs as <opener> "$1" where $1 (also $PR_LAUNCH) is a launcher
# script that cd's into the worktree, runs the command's 'run', then leaves a shell open.
# The opener also gets every $PR_* var below, so a script can name things after the PR,
# or skip $PR_LAUNCH entirely and start its own agent with $PR_WORKTREE + $PR_PROMPT_FILE.
#   Ghostty: open -na Ghostty.app --args -e
#   tmux:    tmux new-window -n "pr-$PR_NUMBER"
#   Herdr:   <gh-prs repo>/openers/herdr.sh   (a space per PR, a tab per command)
#   custom:  ~/bin/my-opener          (any executable)
# A command can set its own 'opener:' to override this one.
opener: open -na Ghostty.app --args -e

# Commands get PR data as env vars, never interpolated: use "$PR_URL", not a template.
#   $PR_NUMBER $PR_REPO $PR_URL $PR_HEAD $PR_BASE $PR_KEY $PR_ACTION
#   $PR_WORKTREE      when worktree: true
#   $PR_PROMPT_FILE   when prompt is set (prompt_context + prompt, rendered with
#                     {{.Number}} {{.Repo}} {{.URL}} {{.Head}} {{.Base}})
#   $PR_SESSION_ID    when background: true (a fresh UUID; pass it to claude --session-id)
#   $PR_AGENT_SESSION when agent_session: true (the PR's last background job)
#   $PR_LAUNCH        when terminal: true (opener only)
# Options: worktree, terminal (hand to the opener), opener (per-command override),
#          background (detached agent job: its stdout must be claude's stream-json, shown
#          live in the agent column), agent_session, interactive (takes over this terminal),
#          confirm (y/N first), refresh (reload PRs after success),
#          timeout (background only: stop the job and everything it started after this
#          long, default 45m, "0" for never; applies even after gh-prs quits)
#
# Background agents run unattended with the tools in --allowedTools (plus your Claude
# settings' allow rules); anything else is denied, not asked. They push when done.
# They use your claude.ai login: env -u drops any ANTHROPIC_API_KEY, which headless claude
# would otherwise use instead. Remove it from run if you want agents billed to an API key.
prompt_context: |
  You're working on GitHub PR #{{.Number}} in {{.Repo}}: {{.URL}}
  Branch {{.Head}} → {{.Base}}. You're in a git worktree on that branch.
  You're running unattended: nobody can answer questions, so make reasonable calls and
  explain them at the end. Use gh to read the PR, its checks and its review comments.
  When you're done, commit and push to {{.Head}} (only that branch).
  Your final message is shown in a small box, so end it with exactly this block and nothing
  after it (no markdown headings; keep each line to one sentence):
    DONE: <one line: what you did>
    - <a change, naming the file in backticks>   (at most 10 lines, the most important first)
    PUSHED: <short commit sha> | NO: <why not>
    NEEDS YOU: <one line: what a person must still decide or check> | nothing

commands:
  - key: cc
    name: get it mergeable
    worktree: true
    background: true
    run: &agent >-
      env -u ANTHROPIC_API_KEY claude -p "$(cat "$PR_PROMPT_FILE")" --session-id "$PR_SESSION_ID"
      --output-format stream-json --verbose --permission-mode acceptEdits
      --allowedTools "Read,Edit,Write,Glob,Grep,Bash(git:*),Bash(gh:*),Bash(npm:*),Bash(npx:*),Bash(pnpm:*),Bash(yarn:*),Bash(node:*),Bash(go:*),Bash(make:*),Bash(python3:*),Bash(pytest:*),Bash(uv:*),Bash(ls:*)"
    prompt: |
      Do what this PR needs to merge: fix failing checks, address unresolved review
      comments, and resolve merge conflicts with {{.Base}}.
  - key: cf
    name: fix CI
    worktree: true
    background: true
    run: *agent
    prompt: |
      Find the failing checks (gh pr checks, then gh run view --log-failed), find the
      root cause and fix it. Run the relevant tests locally before committing.
  - key: ca
    name: address comments
    worktree: true
    background: true
    run: *agent
    prompt: |
      Read the unresolved review threads and address each one. Don't reply to or resolve
      threads; instead, give one change line per thread you addressed.
  - key: cr
    name: rebase
    worktree: true
    background: true
    run: *agent
    prompt: |
      Rebase this branch onto the latest origin/{{.Base}} and resolve conflicts, then push
      with --force-with-lease.
  - key: ct
    name: take over agent
    worktree: true
    terminal: true
    agent_session: true
    run: claude --resume "$PR_AGENT_SESSION"
  - key: w
    name: shell in worktree
    worktree: true
    terminal: true
  # Merge, deploy the title's service(s) to prod with "Deploy Services", then Datadog health
  # checks at +5/+10/+15/+20m. Refuses (before merging) outside the prod deploy window or if
  # anything looks off. Put DEPLOY_DRY_RUN=1 in front to check and plan without merging.
  - key: D
    name: merge + deploy prod
    background: true
    timeout: 3h # backstop: the script itself gives up after 1h merging or 1h deploying
    confirm: true
    refresh: true
    run: '"$GH_PRS_DIR/scripts/deploy-monitor.sh"'
  - key: V
    name: open revert PR
    confirm: true
    refresh: true
    run: >-
      url=$(gh api graphql -f query='mutation($id: ID!) { revertPullRequest(input: {pullRequestId: $id}) { revertPullRequest { url } } }'
      -f id="$(gh pr view "$PR_URL" --json id --jq .id)" --jq .data.revertPullRequest.revertPullRequest.url)
      && gh pr view "$url" --web && echo "$url"
  - key: R
    name: mark ready
    run: gh pr ready "$PR_URL"
    refresh: true
  - key: M
    name: merge
    run: gh pr merge "$PR_URL"
    interactive: true
    confirm: true
    refresh: true
  - key: L
    name: add label
    run: printf 'label to add> ' && read -r l && gh pr edit "$PR_URL" --add-label "$l"
    interactive: true
    refresh: true
  - key: y
    name: copy URL
    run: printf %s "$PR_URL" | pbcopy
  - key: Y
    name: copy branch
    run: printf %s "$PR_HEAD" | pbcopy
`

func configPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "gh-prs", "config.yml")
}

// loadConfig reads the config, writing the default on first run so it is easy to find and edit.
func loadConfig() (Config, error) {
	path := configPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		data = []byte(defaultConfigYAML)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return Config{}, err
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return Config{}, err
		}
	} else if err != nil {
		return Config{}, err
	}
	cfg, err := parseConfig(data)
	if err != nil {
		return cfg, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// parseConfig layers the user's file over the defaults: keys they set replace the default wholesale.
func parseConfig(data []byte) (Config, error) {
	var cfg Config
	if err := yaml.Unmarshal([]byte(defaultConfigYAML), &cfg); err != nil {
		return cfg, fmt.Errorf("default config: %w", err)
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true) // typos and renamed keys fail loudly instead of being ignored
	if err := dec.Decode(&cfg); err != nil && err != io.EOF {
		return cfg, err
	}
	cfg.refresh = 2 * time.Minute
	if cfg.RefreshInterval != "" {
		d, err := time.ParseDuration(cfg.RefreshInterval)
		if cfg.RefreshInterval == "0" {
			d, err = 0, nil
		}
		if err != nil || d < 0 || (d > 0 && d < 30*time.Second) {
			return cfg, fmt.Errorf("refresh_interval %q: use a duration of at least 30s, or 0 to disable", cfg.RefreshInterval)
		}
		cfg.refresh = d
	}
	if len(cfg.Sections) == 0 {
		return cfg, fmt.Errorf("no sections defined")
	}
	for _, s := range cfg.Sections {
		if s.Query == "" {
			return cfg, fmt.Errorf("section %q has no query", s.Title)
		}
		for _, g := range s.GroupBy {
			if _, ok := groupers[g]; !ok {
				return cfg, fmt.Errorf("section %q: unknown group_by %q (use project, status, repo)", s.Title, g)
			}
		}
	}
	if err := validateCommands(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
