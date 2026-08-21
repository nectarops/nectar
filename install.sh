#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-only

set -Eeuo pipefail
IFS=$'\n\t'

readonly PROGRAM="dock-weaver-installer"
readonly DOCKER_KEY_FINGERPRINT="9DC858229FC7DD38854AE2D88D81803C0EBFCD88"
readonly DEFAULT_DOCK_WEAVER_VERSION="0.1.0"
readonly DEFAULT_WEB_PORT="8080"
readonly INSTALL_DIR="/opt/dock-weaver"
readonly DATA_DIR="/var/lib/dock-weaver"
readonly STACK_NAME="dock-weaver"
readonly SECRET_NAME="dock_weaver_init_token"

docker_version=""
advertise_addr=""
web_port="${DEFAULT_WEB_PORT}"
dock_weaver_version="${DEFAULT_DOCK_WEAVER_VERSION}"
force_docker_version=false
dry_run=false

usage() {
  cat <<'EOF'
Install Dock-Weaver on an Ubuntu or Debian Docker Swarm Manager.

Usage: sudo bash install.sh [options]

Options:
  --docker-version VERSION       Install or require this Docker Engine version.
  --advertise-addr ADDRESS      Manager advertise address or interface.
  --web-port PORT               Published Web port (default: 8080).
  --dock-weaver-version VERSION Deploy this pinned Dock-Weaver image tag.
  --force-docker-version        Explicitly allow changing an existing Docker version.
  --dry-run                     Validate and print planned actions without changing the host.
  --help                        Show this help.

Environment:
  DOCK_WEAVER_IMAGE             Override the complete pinned container image reference.
EOF
}

log() {
  printf '[%s] %s\n' "${PROGRAM}" "$*" >&2
}

die() {
  log "ERROR: $*"
  log "Fix the reported condition, then rerun the same install command."
  exit 1
}

on_error() {
  local exit_code=$?
  log "Installation stopped at line ${BASH_LINENO[0]} (exit ${exit_code}). No secret value was logged."
  exit "${exit_code}"
}
trap on_error ERR

run() {
  if [[ "${dry_run}" == true ]]; then
    printf '[dry-run]'
    printf ' %q' "$@"
    printf '\n'
    return 0
  fi
  "$@"
}

while (($# > 0)); do
  case "$1" in
    --docker-version)
      (($# >= 2)) || die "--docker-version requires a value"
      docker_version=$2
      shift 2
      ;;
    --advertise-addr)
      (($# >= 2)) || die "--advertise-addr requires a value"
      advertise_addr=$2
      shift 2
      ;;
    --web-port)
      (($# >= 2)) || die "--web-port requires a value"
      web_port=$2
      shift 2
      ;;
    --dock-weaver-version)
      (($# >= 2)) || die "--dock-weaver-version requires a value"
      dock_weaver_version=$2
      shift 2
      ;;
    --force-docker-version)
      force_docker_version=true
      shift
      ;;
    --dry-run)
      dry_run=true
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      die "unknown argument: $1"
      ;;
  esac
done

[[ "${EUID}" -eq 0 ]] || die "run this installer as root (for example, with sudo)"
[[ "${web_port}" =~ ^[0-9]+$ ]] || die "web port must be numeric"
((web_port >= 1 && web_port <= 65535)) || die "web port must be between 1 and 65535"
[[ "${dock_weaver_version}" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]] ||
  die "Dock-Weaver version must be a pinned semantic version, not latest"
if [[ -n "${docker_version}" ]]; then
  [[ "${docker_version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]] ||
    die "Docker version must look like 29.0.1"
fi
if [[ -n "${advertise_addr}" ]]; then
  [[ "${advertise_addr}" =~ ^[0-9A-Za-z_.:%-]+$ ]] || die "advertise address contains unsupported characters"
fi

[[ -r /etc/os-release ]] || die "/etc/os-release is required"
# shellcheck disable=SC1091
source /etc/os-release
distribution=${ID,,}
case "${distribution}" in
  ubuntu|debian) ;;
  *) die "supported distributions are Ubuntu and Debian; found ${ID:-unknown}" ;;
esac

case "$(uname -m)" in
  x86_64) architecture="amd64" ;;
  aarch64|arm64) architecture="arm64" ;;
  *) die "supported architectures are amd64 and arm64; found $(uname -m)" ;;
esac

if [[ -z "${advertise_addr}" ]]; then
  if command -v ip >/dev/null 2>&1; then
    advertise_addr=$(ip -4 route get 1.1.1.1 2>/dev/null | awk '{for (i=1; i<=NF; i++) if ($i == "src") {print $(i+1); exit}}')
  fi
  [[ -n "${advertise_addr}" ]] || die "unable to determine a Manager address; pass --advertise-addr"
fi

image=${DOCK_WEAVER_IMAGE:-"ghcr.io/ranen/dock-weaver:${dock_weaver_version#v}"}
[[ "${image}" != *":latest" ]] || die "the Dock-Weaver image must not use the latest tag"
[[ "${image}" =~ ^[A-Za-z0-9._:/@-]+$ ]] || die "container image reference contains unsupported characters"

log "Host: ${distribution} ${VERSION_ID:-unknown} (${architecture})"
log "Manager address: ${advertise_addr}; Web port: ${web_port}"
log "Dock-Weaver image: ${image}"

installed_docker=""
if command -v docker >/dev/null 2>&1; then
  installed_docker=$(docker version --format '{{.Server.Version}}' 2>/dev/null || true)
  if [[ -z "${installed_docker}" ]]; then
    die "a Docker CLI is installed but its daemon is unavailable; start or repair Docker before rerunning so the installer cannot change an unknown installation"
  fi
fi

if [[ -n "${installed_docker}" ]]; then
  log "Detected Docker Engine ${installed_docker}."
  if [[ -n "${docker_version}" && "${installed_docker}" != "${docker_version}" ]]; then
    [[ "${force_docker_version}" == true ]] || die \
      "Docker ${installed_docker} is installed but ${docker_version} was requested; rerun with --force-docker-version after reviewing upgrade or downgrade risk"
  else
    docker_version=${docker_version:-${installed_docker}}
  fi
fi

install_docker() {
  local keyring="/etc/apt/keyrings/docker.asc"
  local repo_arch
  local codename
  local package_version=""
  local fingerprint

  repo_arch=$(dpkg --print-architecture)
  [[ "${repo_arch}" == "${architecture}" ]] || die "dpkg architecture ${repo_arch} does not match host ${architecture}"
  codename=${VERSION_CODENAME:-}
  [[ -n "${codename}" ]] || die "VERSION_CODENAME is missing from /etc/os-release"

  run apt-get update
  run apt-get install -y ca-certificates curl gpg
  run install -m 0755 -d /etc/apt/keyrings

  if [[ "${dry_run}" == true ]]; then
    log "Would download and verify Docker's repository signing key fingerprint ${DOCKER_KEY_FINGERPRINT}."
  else
    curl -fsSL "https://download.docker.com/linux/${distribution}/gpg" -o "${keyring}.tmp"
    fingerprint=$(gpg --show-keys --with-colons "${keyring}.tmp" | awk -F: '$1 == "fpr" {print $10; exit}')
    [[ "${fingerprint}" == "${DOCKER_KEY_FINGERPRINT}" ]] || {
      rm -f "${keyring}.tmp"
      die "Docker repository signing-key fingerprint did not match"
    }
    install -m 0644 "${keyring}.tmp" "${keyring}"
    rm -f "${keyring}.tmp"
  fi

  if [[ "${dry_run}" == true ]]; then
    log "Would configure Docker's signed ${distribution} repository for ${codename}."
  else
    printf 'deb [arch=%s signed-by=%s] https://download.docker.com/linux/%s %s stable\n' \
      "${repo_arch}" "${keyring}" "${distribution}" "${codename}" > /etc/apt/sources.list.d/docker.list
  fi
  run apt-get update

  if [[ "${dry_run}" != true ]]; then
    if [[ -n "${docker_version}" ]]; then
      package_version=$(apt-cache madison docker-ce | awk -v requested="${docker_version}" '$3 ~ ("^5:" requested "-") {print $3; exit}')
      [[ -n "${package_version}" ]] || die "Docker ${docker_version} is not available for this distribution"
    else
      package_version=$(apt-cache madison docker-ce | awk 'NR == 1 {print $3}')
      [[ -n "${package_version}" ]] || die "Docker's repository did not provide docker-ce"
      docker_version=${package_version#5:}
      docker_version=${docker_version%%-*}
    fi
    apt-get install -y --allow-downgrades \
      "docker-ce=${package_version}" \
      "docker-ce-cli=${package_version}" \
      containerd.io docker-buildx-plugin docker-compose-plugin
    apt-mark hold docker-ce docker-ce-cli
  else
    log "Would install and hold Docker Engine ${docker_version:-the current repository version}."
  fi
}

if [[ -z "${installed_docker}" || ( -n "${docker_version}" && "${installed_docker}" != "${docker_version}" ) ]]; then
  install_docker
fi

run systemctl enable --now docker
if [[ "${dry_run}" != true ]]; then
  docker info >/dev/null
  actual_docker=$(docker version --format '{{.Server.Version}}')
  [[ -z "${docker_version}" || "${actual_docker}" == "${docker_version}" ]] ||
    die "Docker version verification failed: expected ${docker_version}, found ${actual_docker}"
fi

swarm_state="inactive"
if [[ "${dry_run}" != true ]]; then
  swarm_state=$(docker info --format '{{.Swarm.LocalNodeState}}')
fi
case "${swarm_state}" in
  inactive)
    run docker swarm init --advertise-addr "${advertise_addr}"
    ;;
  active)
    if [[ "${dry_run}" != true ]]; then
      control_available=$(docker info --format '{{.Swarm.ControlAvailable}}')
      [[ "${control_available}" == "true" ]] || die "this host already belongs to a Swarm as a Worker; it will not be removed or promoted automatically"
    fi
    log "This host is already an active Swarm Manager; preserving its membership."
    ;;
  pending|locked|error)
    die "Docker reports Swarm state ${swarm_state}; resolve it before installing"
    ;;
  *) die "unexpected Swarm state: ${swarm_state}" ;;
esac

if [[ "${dry_run}" != true ]]; then
  manager_node_id=$(docker info --format '{{.Swarm.NodeID}}')
  docker node update --label-add dock-weaver.control=true "${manager_node_id}" >/dev/null
else
  log "Would label the current Manager node dock-weaver.control=true."
fi

service_exists=false
if [[ "${dry_run}" != true ]] && docker service inspect "${STACK_NAME}_dock-weaver" >/dev/null 2>&1; then
  service_exists=true
fi
if [[ "${service_exists}" != true ]] && command -v ss >/dev/null 2>&1 &&
  ss -H -ltn "sport = :${web_port}" 2>/dev/null | grep -q .; then
  die "TCP port ${web_port} is already listening; select another value with --web-port"
fi

run install -d -m 0700 "${DATA_DIR}"
run install -d -m 0755 "${INSTALL_DIR}"
token_file="${DATA_DIR}/bootstrap-token"
if [[ "${dry_run}" != true && ! -s "${token_file}" ]]; then
  umask 077
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -base64 32 | tr -d '\n' > "${token_file}"
  else
    head -c 32 /dev/urandom | base64 | tr -d '\n' > "${token_file}"
  fi
  chmod 0600 "${token_file}"
fi

if [[ "${dry_run}" != true ]] && ! docker secret inspect "${SECRET_NAME}" >/dev/null 2>&1; then
  docker secret create "${SECRET_NAME}" "${token_file}" >/dev/null
elif [[ "${dry_run}" == true ]]; then
  log "Would create the external ${SECRET_NAME} secret if absent."
fi

stack_file="${INSTALL_DIR}/stack.yml"
if [[ "${dry_run}" == true ]]; then
  log "Would write the pinned Swarm stack to ${stack_file}."
else
  cat > "${stack_file}" <<EOF
version: "3.9"
services:
  dock-weaver:
    image: "${image}"
    user: root
    environment:
      DW_ADDR: ":8080"
      DW_COOKIE_SECURE: "false"
      DW_DATA_DIR: /var/lib/dock-weaver
      DW_INIT_TOKEN_FILE: /run/secrets/${SECRET_NAME}
      DW_REQUIRE_DOCKER: "true"
    ports:
      - target: 8080
        published: ${web_port}
        protocol: tcp
        mode: ingress
    volumes:
      - dock_weaver_data:/var/lib/dock-weaver
      - /var/run/docker.sock:/var/run/docker.sock
    secrets:
      - ${SECRET_NAME}
    deploy:
      replicas: 1
      placement:
        constraints:
          - node.role == manager
          - node.labels.dock-weaver.control == true
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 5
      update_config:
        parallelism: 1
        order: stop-first
        failure_action: rollback
volumes:
  dock_weaver_data:
secrets:
  ${SECRET_NAME}:
    external: true
EOF
  chmod 0644 "${stack_file}"
fi

run docker stack deploy --detach=true "${STACK_NAME}" --compose-file "${stack_file}"

setup_url="http://${advertise_addr}:${web_port}/"
if [[ "${dry_run}" != true ]]; then
  log "Waiting for Dock-Weaver readiness at ${setup_url}health/ready …"
  ready=false
  for _ in $(seq 1 60); do
    if curl -fsS --max-time 3 "${setup_url}health/ready" >/dev/null 2>&1; then
      ready=true
      break
    fi
    sleep 2
  done
  [[ "${ready}" == true ]] || die "Dock-Weaver did not become ready; inspect: docker service ps ${STACK_NAME}_dock-weaver"

  printf '\nDock-Weaver is ready.\n'
  printf 'Setup URL: %s\n' "${setup_url}"
  printf 'One-time setup token: %s\n' "$(<"${token_file}")"
  printf 'The token file is root-readable at %s for safe installer resume. Delete it after setup.\n' "${token_file}"
else
  printf '\nDry run completed. No host changes were made.\n'
  printf 'Planned setup URL: %s\n' "${setup_url}"
fi
