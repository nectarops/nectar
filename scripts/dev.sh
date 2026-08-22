#!/bin/sh
# SPDX-License-Identifier: AGPL-3.0-only

set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"

address=${DW_ADDR:-127.0.0.1:8080}
data_dir=${DW_DATA_DIR:-${TMPDIR:-/tmp}/dock-weaver-dev}
require_docker=${DW_REQUIRE_DOCKER:-false}
cookie_secure=${DW_COOKIE_SECURE:-false}
build=true
open_browser=true

usage() {
	cat <<'EOF'
Start Dock-Weaver locally with one command.

Usage:
  ./scripts/dev.sh [options]

Options:
  --addr ADDRESS       Listen address (default: 127.0.0.1:8080)
  --data-dir PATH      Runtime data directory (default: system temp directory)
  --require-docker     Fail startup unless the Docker Engine is available
  --skip-build         Reuse the existing bin/dock-weaver binary
  --no-open            Do not open the web page automatically
  -h, --help           Show this help

Environment variables:
  DW_ADDR, DW_DATA_DIR, DW_INIT_TOKEN, DW_INIT_TOKEN_FILE,
  DW_COOKIE_SECURE, and DW_REQUIRE_DOCKER override the defaults.
EOF
}

die() {
	printf 'error: %s\n' "$*" >&2
	exit 1
}

require_command() {
	command -v "$1" >/dev/null 2>&1 || die "$1 is required but was not found in PATH"
}

require_value() {
	option=$1
	value=${2:-}
	[ -n "$value" ] || die "$option requires a value"
}

while [ "$#" -gt 0 ]; do
	case "$1" in
	--addr)
		require_value "$1" "${2:-}"
		address=$2
		shift 2
		;;
	--data-dir)
		require_value "$1" "${2:-}"
		data_dir=$2
		shift 2
		;;
	--require-docker)
		require_docker=true
		shift
		;;
	--skip-build)
		build=false
		shift
		;;
	--no-open)
		open_browser=false
		shift
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		printf 'error: unknown option: %s\n\n' "$1" >&2
		usage >&2
		exit 2
		;;
	esac
done

case "$require_docker" in
true | false) ;;
*) die "DW_REQUIRE_DOCKER must be true or false" ;;
esac

case "$cookie_secure" in
true | false) ;;
*) die "DW_COOKIE_SECURE must be true or false" ;;
esac

mkdir -p "$data_dir"

token=${DW_INIT_TOKEN:-}
token_file=${DW_INIT_TOKEN_FILE:-}
if [ -z "$token" ]; then
	if [ -z "$token_file" ]; then
		token_file=$data_dir/bootstrap-token
	fi

	if [ ! -s "$token_file" ]; then
		umask 077
		if command -v openssl >/dev/null 2>&1; then
			openssl rand -hex 24 >"$token_file"
		else
			require_command od
			require_command tr
			od -An -N24 -tx1 /dev/urandom | tr -d ' \n' >"$token_file"
		fi
	fi

	chmod 600 "$token_file"
	token=$(sed -n '1p' "$token_file")
	[ -n "$token" ] || die "initialization token file is empty: $token_file"
fi

if [ "$require_docker" = true ]; then
	require_command docker
	docker info >/dev/null 2>&1 || die "Docker Engine is not available"
fi

if [ "$build" = true ]; then
	require_command make
	require_command go
	require_command node
	require_command pnpm
	printf '%s\n' 'Installing pinned web dependencies...'
	make install-web
	printf '%s\n' 'Building the web application and Go server...'
	make build
elif [ ! -x ./bin/dock-weaver ]; then
	die "bin/dock-weaver does not exist; run without --skip-build first"
fi

case "$address" in
:*) browser_address=127.0.0.1$address ;;
0.0.0.0:*) browser_address=127.0.0.1:${address#0.0.0.0:} ;;
*) browser_address=$address ;;
esac
web_url=http://$browser_address

commit=dev
if command -v git >/dev/null 2>&1; then
	commit=$(git rev-parse --short HEAD 2>/dev/null || printf '%s' dev)
fi

export DW_ADDR="$address"
export DW_DATA_DIR="$data_dir"
export DW_INIT_TOKEN="$token"
export DW_COOKIE_SECURE="$cookie_secure"
export DW_REQUIRE_DOCKER="$require_docker"
export DW_VERSION="${DW_VERSION:-dev}"
export DW_COMMIT="${DW_COMMIT:-$commit}"

printf '\nDock-Weaver is starting.\n'
printf '  Web:          %s\n' "$web_url"
printf '  Data:         %s\n' "$data_dir"
printf '  Setup token:  %s\n' "$token"
printf '  Docker:       required=%s\n\n' "$require_docker"

if [ "$open_browser" = true ]; then
	opener=
	if command -v open >/dev/null 2>&1; then
		opener=open
	elif command -v xdg-open >/dev/null 2>&1; then
		opener=xdg-open
	fi

	if [ -n "$opener" ]; then
		(
			if command -v curl >/dev/null 2>&1; then
				attempt=0
				while [ "$attempt" -lt 30 ]; do
					if curl -fsS "$web_url/health/live" >/dev/null 2>&1; then
						"$opener" "$web_url" >/dev/null 2>&1 || true
						exit 0
					fi
					attempt=$((attempt + 1))
					sleep 1
				done
			else
				sleep 2
				"$opener" "$web_url" >/dev/null 2>&1 || true
			fi
		) &
	fi
fi

exec ./bin/dock-weaver
