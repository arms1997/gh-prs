#!/bin/sh
# gh-prs opener for the Herdr app: one space per PR (named "<repo> #<n>", first tab a shell),
# and a new tab per command, named after it ("claude fix CI"), running the gh-prs launcher.
# Use it with:  opener: /path/to/openers/herdr.sh
# Needs herdr-app and jq on PATH.
set -eu

launch=$1
name="$(basename "$PR_REPO") #$PR_NUMBER"
folder=${PR_WORKTREE:-$PWD}

space=$(herdr-app ls | jq -r --arg n "$name" '[.spaces[] | select(.name == $n and .machine == null)][0].id // empty')
if [ -z "$space" ]; then
  space=$(herdr-app do space.new "$(jq -cn --arg n "$name" --arg f "$folder" '{name: $n, folder: $f, tab: "shell"}')" | jq -r .space)
fi

tab=$(herdr-app do tab.new "$(jq -cn --arg s "$space" --arg n "$PR_ACTION" --arg f "$folder" --arg l "$launch" \
  '{space: $s, name: $n, folder: $f, argv: ["sh", $l]}')" | jq -r .tab)
herdr-app do space.select "$(jq -cn --arg i "$space" '{id: $i}')" >/dev/null
herdr-app do tab.select "$(jq -cn --arg i "$tab" '{id: $i}')" >/dev/null
