#!/bin/sh
# SPDX-License-Identifier: AGPL-3.0-only

set -eu

program=nectar-release
project_root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"

die() {
  printf '%s: %s\n' "$program" "$*" >&2
  exit 1
}

[ "$#" -eq 1 ] || die 'usage: ./scripts/create-release-tag.sh VERSION'
version=${1#v}
printf '%s\n' "$version" |
  grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$' ||
  die "invalid semantic version: $1"
tag="v$version"

git rev-parse --is-inside-work-tree >/dev/null 2>&1 || die 'not inside a Git worktree'
git symbolic-ref --quiet --short HEAD >/dev/null 2>&1 ||
  die 'release tags must be created from a branch, not a detached HEAD'
dirty_status=$(git status --short)
if [ -n "$dirty_status" ]; then
  printf '%s\n' "$program: commit the existing feature changes before preparing the release version:" >&2
  printf '%s\n' "$dirty_status" >&2
  cat >&2 <<'EOF'

The required order is:
  1. Commit the existing feature changes.
  2. Run make release-tag VERSION=x.y.z from the clean worktree.
  3. The command updates the version, verifies it, creates the release commit, and then creates the tag.
EOF
  exit 1
fi
if git rev-parse --verify --quiet "refs/tags/$tag" >/dev/null; then
  die "tag $tag already exists"
fi

./scripts/set-version.sh "$version"
make verify
git diff --check
unexpected_changes=$(
  {
    git diff --name-only
    git ls-files --others --exclude-standard
  } | awk '$0 != "install.sh" && $0 != "README.md"'
)
[ -z "$unexpected_changes" ] ||
  die "verification changed unexpected files; review before tagging:
$unexpected_changes"
git add install.sh README.md
if git diff --cached --quiet; then
  printf '%s: version files already match %s; tagging the current commit\n' "$program" "$version"
else
  git commit -m "chore(release): prepare $tag"
fi
git tag -a "$tag" -m "$tag"

cat <<EOF
$program: created annotated tag $tag

Review the tag, then publish it explicitly:
  git push origin HEAD
  git push origin $tag
EOF
