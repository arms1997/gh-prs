#!/bin/sh
# Tags a merged PR and publishes a release with the gh extension binaries:
#   scripts/release.sh <PR number>
# The version goes up by the PR's label: `major`, `minor`, or a patch by default. Run it after a
# merge (`make release PR=12`), or let ci/workflows/release.yml run it once CI is enabled.
# RELEASE_DRY_RUN=1 builds and prints the version without tagging or publishing.
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
if git tag --points-at "$sha" | grep -q '^v'; then
	echo "already released: $(git tag --points-at "$sha" | grep '^v')"
	exit 0
fi
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
for target in darwin/arm64 darwin/amd64 linux/amd64 linux/arm64; do
	(cd "$dist/src" && CGO_ENABLED=0 GOOS=${target%/*} GOARCH=${target#*/} \
		go build -trimpath -ldflags "-s -w" -o "$dist/gh-prs_${tag}_${target%/*}-${target#*/}" .)
done
git worktree remove --force "$dist/src"

if [ -n "${RELEASE_DRY_RUN:-}" ]; then
	echo "dry run: would release $tag from $sha with:"
	ls "$dist"/gh-prs_*
	exit 0
fi
git tag "$tag" "$sha"
git push -q origin "$tag"
gh release create "$tag" -R "$repo" "$dist"/gh-prs_* --title "$tag" --notes "$title (#$pr)"
echo "released $tag"
