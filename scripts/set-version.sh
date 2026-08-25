#!/bin/sh
# SPDX-License-Identifier: AGPL-3.0-only

set -eu

program=nectar-set-version
project_root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"

usage() {
  cat <<'EOF'
Update or verify the release version used by install.sh and README.md.

Usage:
  ./scripts/set-version.sh VERSION
  ./scripts/set-version.sh --check [VERSION]

VERSION accepts 1.2.3 or v1.2.3. The stored form never includes the v prefix.
EOF
}

die() {
  printf '%s: %s\n' "$program" "$*" >&2
  exit 1
}

normalize_version() {
  version=${1#v}
  case "$version" in
    '' | *[!0-9A-Za-z.-]*) return 1 ;;
  esac
  printf '%s\n' "$version" |
    grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$' || return 1
  printf '%s\n' "$version"
}

installer_version() {
  versions=$(sed -n 's/^readonly DEFAULT_NECTAR_VERSION="\([^"]*\)"$/\1/p' install.sh)
  count=$(printf '%s\n' "$versions" | awk 'NF {count++} END {print count + 0}')
  [ "$count" -eq 1 ] || die 'install.sh must define DEFAULT_NECTAR_VERSION exactly once'
  printf '%s\n' "$versions"
}

check_readme_version() {
  expected=$1
  awk -v expected="$expected" '
		BEGIN {
			failed = 0
			marker = 0
			download = 0
			flag = 0
			image = 0
		}
		/<!-- nectar-release-version:/ {
			marker++
			if (index($0, "<!-- nectar-release-version: " expected " -->") == 0) {
				print "README release-version marker does not match " expected > "/dev/stderr"
				failed = 1
			}
		}
		/releases\/download\/v[0-9]/ {
			download++
			if (index($0, "/v" expected "/") == 0) {
				print "README release download URL does not match " expected ": " $0 > "/dev/stderr"
				failed = 1
			}
		}
		/--nectar-version [v0-9]/ {
			flag++
			if (index($0, "--nectar-version " expected) == 0) {
				print "README --nectar-version example does not match " expected ": " $0 > "/dev/stderr"
				failed = 1
			}
		}
		/NECTAR_IMAGE=.*nectar:[v0-9]/ {
			image++
			if (index($0, "nectar:" expected) == 0) {
				print "README image example does not match " expected ": " $0 > "/dev/stderr"
				failed = 1
			}
		}
		END {
			if (marker != 1) {
				print "README must contain exactly one nectar-release-version marker" > "/dev/stderr"
				failed = 1
			}
			if (download == 0 || flag == 0 || image == 0) {
				print "README release examples are incomplete" > "/dev/stderr"
				failed = 1
			}
			exit failed
		}
	' README.md
}

check_version() {
  expected=$1
  actual=$(installer_version)
  [ "$actual" = "$expected" ] ||
    die "install.sh uses $actual but expected $expected"
  check_readme_version "$expected" ||
    die "README.md does not consistently use $expected"
}

mode=update
case "${1:-}" in
  --check)
    mode=check
    shift
    ;;
  --help | -h)
    usage
    exit 0
    ;;
esac

case "$#" in
  0)
    [ "$mode" = check ] || {
      usage >&2
      exit 2
    }
    target=$(installer_version)
    ;;
  1)
    target=$(normalize_version "$1") || die "invalid semantic version: $1"
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac

if [ "$mode" = check ]; then
  check_version "$target"
  printf '%s: release version %s is consistent\n' "$program" "$target"
  exit 0
fi

current=$(installer_version)
check_version "$current"
if [ "$current" = "$target" ]; then
  printf '%s: release version is already %s\n' "$program" "$target"
  exit 0
fi

command -v perl >/dev/null 2>&1 || die 'perl is required to update release version files'
NECTAR_OLD_VERSION=$current NECTAR_NEW_VERSION=$target perl -0pi -e '
	s/\Q$ENV{NECTAR_OLD_VERSION}\E/$ENV{NECTAR_NEW_VERSION}/g
' install.sh README.md

check_version "$target"
printf '%s: updated release version %s -> %s in install.sh and README.md\n' \
  "$program" "$current" "$target"
