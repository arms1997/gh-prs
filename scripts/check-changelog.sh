#!/bin/sh
# A branch that changes what users get (the app, the terminal UI, core, built-in config, prompts,
# plugin, openers, deploy script) must add a line under "## Unreleased" in CHANGELOG.md. Tests and
# docs don't need one. Runs before every push (scripts/hooks/pre-push); `git push --no-verify` skips.
set -eu
base=$(git merge-base HEAD origin/main 2>/dev/null) || exit 0
user_facing=$(git diff --name-only "$base" HEAD |
	grep -E '^(core/|desktop/|defaults/|skills/|openers/|scripts/deploy-monitor\.sh$|[^/]+\.go$|go\.mod$)' |
	grep -vE '(_test\.go$|\.test\.tsx?$|/testdata/|/src/test/)' || true)
[ -n "$user_facing" ] || exit 0

# The lines of the Unreleased section, at a commit.
unreleased() { git show "$1:CHANGELOG.md" 2>/dev/null | awk '/^## /{on = ($0 == "## Unreleased")} on && /^- /' | sort; }
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
unreleased "$base" >"$tmp/before"
unreleased HEAD >"$tmp/after"
if [ -z "$(comm -13 "$tmp/before" "$tmp/after")" ]; then
	echo "CHANGELOG.md: add a line under ## Unreleased; this branch changes what users get:" >&2
	printf '  %s\n' $user_facing >&2
	exit 1
fi
