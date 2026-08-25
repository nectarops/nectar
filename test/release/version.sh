#!/bin/sh
# SPDX-License-Identifier: AGPL-3.0-only

set -eu

project_root=$(CDPATH='' cd -- "$(dirname -- "$0")/../.." && pwd)
test_root=$(mktemp -d "${TMPDIR:-/tmp}/nectar-release-test.XXXXXX")

cleanup() {
  rm -rf -- "$test_root"
}
trap cleanup EXIT HUP INT TERM

mkdir -p "$test_root/scripts"
cp "$project_root/install.sh" "$test_root/install.sh"
cp "$project_root/README.md" "$test_root/README.md"
cp "$project_root/scripts/set-version.sh" "$test_root/scripts/set-version.sh"
cp "$project_root/scripts/create-release-tag.sh" "$test_root/scripts/create-release-tag.sh"
chmod +x "$test_root/scripts/set-version.sh" "$test_root/scripts/create-release-tag.sh"

"$test_root/scripts/set-version.sh" --check >/dev/null
"$test_root/scripts/set-version.sh" v9.8.7 >/dev/null
"$test_root/scripts/set-version.sh" --check 9.8.7 >/dev/null

grep -F 'readonly DEFAULT_NECTAR_VERSION="9.8.7"' "$test_root/install.sh" >/dev/null
grep -F '<!-- nectar-release-version: 9.8.7 -->' "$test_root/README.md" >/dev/null
if "$test_root/scripts/set-version.sh" --check 9.8.6 >/dev/null 2>&1; then
  printf '%s\n' 'version check accepted the wrong expected version' >&2
  exit 1
fi
if "$test_root/scripts/set-version.sh" invalid >/dev/null 2>&1; then
  printf '%s\n' 'version update accepted an invalid semantic version' >&2
  exit 1
fi

cat >"$test_root/Makefile" <<'EOF'
.PHONY: verify
verify:
	./scripts/set-version.sh --check
EOF

git -C "$test_root" init -q
git -C "$test_root" config user.name 'Nectar Release Test'
git -C "$test_root" config user.email 'release-test@example.invalid'
git -C "$test_root" add .
git -C "$test_root" commit -qm 'test: initialize release fixture'

printf '%s\n' 'uncommitted feature work' >"$test_root/dirty.txt"
if "$test_root/scripts/create-release-tag.sh" 9.8.8 >/dev/null 2>&1; then
  printf '%s\n' 'release tag creation accepted a dirty worktree' >&2
  exit 1
fi
rm "$test_root/dirty.txt"

"$test_root/scripts/create-release-tag.sh" 9.8.8 >/dev/null
[ "$(git -C "$test_root" tag --list v9.8.8)" = v9.8.8 ]
[ "$(git -C "$test_root" log -1 --format=%s)" = 'chore(release): prepare v9.8.8' ]
"$test_root/scripts/set-version.sh" --check 9.8.8 >/dev/null
[ -z "$(git -C "$test_root" status --porcelain)" ]

"$test_root/scripts/create-release-tag.sh" 9.8.9 >/dev/null
[ "$(git -C "$test_root" tag --list v9.8.9)" = v9.8.9 ]
[ "$(git -C "$test_root" log -1 --format=%s)" = 'chore(release): prepare v9.8.9' ]
[ -z "$(git -C "$test_root" status --porcelain)" ]

printf '%s\n' 'release version automation tests passed'
