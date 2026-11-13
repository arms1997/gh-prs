#!/bin/sh
# Tags a merged PR and publishes a release with the gh extension binaries:
#   scripts/release.sh <PR number>
# The version goes up by the PR's label: `major`, `minor`, or a patch by default. Run it right after
# the merge (`make release PR=12`), or let ci/workflows/release.yml run it once CI is enabled.
# CHANGELOG.md's Unreleased section becomes the release notes, and a release commit on main renames
# it to the version; the tag goes on that commit. Pushing to main needs a ruleset bypass (you, as admin).
# RELEASE_DRY_RUN=1 builds and prints the version, notes and changelog change without publishing.
set -eu
pr=${1:?usage: scripts/release.sh <PR number>}
repo=${GITHUB_REPOSITORY:-arms1997/gh-prs}

info=$(gh pr view "$pr" -R "$repo" --json state,mergeCommit,labels,title \
	--jq '[.state, .mergeCommit.oid, ([.labels[].name] | join(" ")), .title] | @tsv')
state=$(printf '%s' "$info" | cut -f1)
sha=$(printf '%s' "$info" | cut -f2)
labels=$(printf '%s' "$info" | cut -f3)
title=$(printf '%s' "$info" | cut -f4)
[ "$state" = MERGED ] || { echo "PR #$pr isn't merged ($state)" >&2; exit 1; }

git fetch -q --tags origin
if released=$(git tag --list 'v*' --contains "$sha" | head -1) && [ -n "$released" ]; then
	echo "already released in $released"
	exit 0
fi
# The release commit goes on top of the merge, so nothing else may have merged since.
[ "$(git rev-parse origin/main)" = "$sha" ] || {
	echo "main has moved past PR #$pr's merge; release each PR right after it merges" >&2
	exit 1
}
last=$(git tag --list 'v*' --sort=-v:refname | head -1)
last=${last:-v0.0.0}
IFS=. read -r major minor patch <<EOT
${last#v}
EOT
case " $labels " in
*" major "*) major=$((major + 1)) minor=0 patch=0 ;;
*" minor "*) minor=$((minor + 1)) patch=0 ;;
*) patch=$((patch + 1)) ;;
esac
tag="v$major.$minor.$patch"

dist=$(mktemp -d)
trap 'rm -rf "$dist"' EXIT
git worktree add -q --detach "$dist/src" "$sha"
src=$dist/src
notes=$(awk '/^## /{on = ($0 == "## Unreleased"); next} on' "$src/CHANGELOG.md" | sed '/./,$!d')
if [ -n "$notes" ]; then
	awk -v head="## $tag — $(date +%Y-%m-%d)" '{print} $0 == "## Unreleased" {print ""; print head}' \
		"$src/CHANGELOG.md" >"$dist/CHANGELOG.md"
	mv "$dist/CHANGELOG.md" "$src/CHANGELOG.md"
	if [ -z "${RELEASE_DRY_RUN:-}" ]; then
		git -C "$src" commit -q -m "chore: release $tag" -- CHANGELOG.md
		sha=$(git -C "$src" rev-parse HEAD)
	fi
else
	notes="$title (#$pr)"
fi
printf '%s\n\n%s\n' "$notes" "PR: https://github.com/$repo/pull/$pr" >"$dist/notes.md"

for target in darwin/arm64 darwin/amd64 linux/amd64 linux/arm64; do
	(cd "$src" && CGO_ENABLED=0 GOOS=${target%/*} GOARCH=${target#*/} \
		go build -trimpath -ldflags "-s -w" -o "$dist/gh-prs_${tag}_${target%/*}-${target#*/}" .)
done

if [ -n "${RELEASE_DRY_RUN:-}" ]; then
	echo "dry run: would release $tag with:"
	ls "$dist"/gh-prs_*
	echo "--- notes"
	cat "$dist/notes.md"
	echo "--- CHANGELOG.md"
	git -C "$src" diff -- CHANGELOG.md
	git worktree remove --force "$src"
	exit 0
fi
git worktree remove --force "$src"
# The release commit only touches CHANGELOG.md; the merge it sits on already passed make check.
git push -q --no-verify origin "$sha:refs/heads/main"
git tag "$tag" "$sha"
git push -q --no-verify origin "$tag"
gh release create "$tag" -R "$repo" "$dist"/gh-prs_* --title "$tag" --notes-file "$dist/notes.md"
echo "released $tag"
