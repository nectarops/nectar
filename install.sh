#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-only

set -Eeuo pipefail
IFS=$'\n\t'

readonly PROGRAM="nectar-installer"
readonly DOCKER_KEY_FINGERPRINT="9DC858229FC7DD38854AE2D88D81803C0EBFCD88"
readonly DEFAULT_DOCKER_REPOSITORY_URL="https://download.docker.com/linux"
readonly DEFAULT_NECTAR_VERSION="0.1.1"
readonly DEFAULT_WEB_PORT="8080"
readonly INSTALL_DIR="/opt/nectar"
readonly DATA_DIR="/var/lib/nectar"
readonly STACK_NAME="nectar"
readonly SECRET_NAME="nectar_init_token"
readonly DAEMON_CONFIG="/etc/docker/daemon.json"

docker_version=""
advertise_addr=""
web_port="${DEFAULT_WEB_PORT}"
nectar_version="${DEFAULT_NECTAR_VERSION}"
force_docker_version=false
dry_run=false

docker_config_changed=false
usage() {
  cat <<'EOF'
Install Nectar on an Ubuntu or Debian Docker Swarm Manager.

Usage: sudo bash install.sh [options]

Options:
  --docker-version VERSION       Install or require this Docker Engine version.
  --advertise-addr ADDRESS       Manager advertise address or interface.
  --web-port PORT               Published Web port (default: 8080).
  --nectar-version VERSION       Deploy this pinned Nectar image tag.
  --force-docker-version        Explicitly allow changing an existing Docker version.
  --dry-run                     Validate and print planned actions without changing the host.
  --help                        Show this help.

Environment:
  NECTAR_IMAGE                  Override the complete pinned container image reference.
  NECTAR_DOCKER_REPOSITORY_URL  Override the Docker CE repository root.
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
    --nectar-version)
      (($# >= 2)) || die "--nectar-version requires a value"
      nectar_version=$2
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
[[ "${nectar_version}" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]] ||
  die "Nectar version must be a pinned semantic version, not latest"
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

image=${NECTAR_IMAGE:-"ghcr.io/nectarops/nectar:${nectar_version#v}"}
[[ "${image}" != *":latest" ]] || die "the Nectar image must not use the latest tag"
[[ "${image}" =~ ^[A-Za-z0-9._:/@-]+$ ]] || die "container image reference contains unsupported characters"

docker_repository_url=${NECTAR_DOCKER_REPOSITORY_URL:-"${DEFAULT_DOCKER_REPOSITORY_URL}"}
docker_repository_url=${docker_repository_url%/}
[[ "${docker_repository_url}" =~ ^https://[A-Za-z0-9.-]+(:[0-9]+)?(/[A-Za-z0-9._/-]+)*$ ]] ||
  die "Docker repository URL must be an HTTPS URL without query parameters"

log "Host: ${distribution} ${VERSION_ID:-unknown} (${architecture})"
log "Manager address: ${advertise_addr}; Web port: ${web_port}"
log "Nectar image: ${image}"
log "Docker repository: ${docker_repository_url}"

installed_docker=""
actual_docker=""
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
    if ! curl --retry 5 --retry-all-errors --connect-timeout 15 -fsSL \
      "${docker_repository_url}/${distribution}/gpg" -o "${keyring}.tmp"; then
      rm -f "${keyring}.tmp"
      die "unable to download Docker's signing key from ${docker_repository_url}; check outbound HTTPS or set NECTAR_DOCKER_REPOSITORY_URL to a trusted mirror"
    fi
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
    printf 'deb [arch=%s signed-by=%s] %s/%s %s stable\n' \
      "${repo_arch}" "${keyring}" "${docker_repository_url}" "${distribution}" "${codename}" > /etc/apt/sources.list.d/docker.list
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

ensure_json_tool() {
  if command -v jq >/dev/null 2>&1; then
    return
  fi
  run apt-get update
  run apt-get install -y jq
  if [[ "${dry_run}" != true ]]; then
    command -v jq >/dev/null 2>&1 || die "jq is required to merge ${DAEMON_CONFIG} safely"
  fi
}

protect_manager_quorum() {
  [[ "${dry_run}" != true && -n "${installed_docker}" ]] || return 0
  [[ "$(docker info --format '{{.Swarm.LocalNodeState}}')" == "active" ]] || return 0
  [[ "$(docker info --format '{{.Swarm.ControlAvailable}}')" == "true" ]] || return 0

  local managers
  local quorum
  local reachable
  managers=$(docker node ls --filter role=manager --format '{{.ID}}' | wc -l | tr -d ' ')
  reachable=$(docker node ls --filter role=manager --format '{{.ManagerStatus}}' |
    awk '$0 == "Leader" || $0 == "Reachable" {count++} END {print count+0}')
  if ((managers == 1)); then
    log "Docker restart will briefly interrupt this single-Manager control plane."
    return
  fi
  quorum=$((managers / 2 + 1))
  ((reachable - 1 >= quorum)) ||
    die "refusing to restart Docker: the remaining reachable Managers would not preserve quorum"
}

configure_docker_logging() {
  if [[ "${dry_run}" == true ]]; then
    log "Would safely merge json-file rotation settings into ${DAEMON_CONFIG}, validate it, and restart Docker only if changed."
    docker_config_changed=true
    return
  fi

  ensure_json_tool
  install -d -m 0755 "$(dirname "${DAEMON_CONFIG}")"
  local candidate
  candidate=$(mktemp "${DAEMON_CONFIG}.tmp.XXXXXX")

  if [[ -e "${DAEMON_CONFIG}" ]]; then
    jq -e '
      type == "object" and
      (.["log-opts"] == null or (.["log-opts"] | type == "object"))
    ' "${DAEMON_CONFIG}" >/dev/null ||
      die "${DAEMON_CONFIG} and its log-opts value must contain JSON objects"
    if ! jq '.
      | .["log-driver"] = "json-file"
      | .["log-opts"] = ((.["log-opts"] // {}) + {
          "max-size": "100m",
          "max-file": "3",
          "compress": "true"
        })' "${DAEMON_CONFIG}" >"${candidate}"; then
      rm -f "${candidate}"
      die "unable to merge Docker logging settings"
    fi
  else
    printf '{}\n' |
      jq '.
        | .["log-driver"] = "json-file"
        | .["log-opts"] = {
            "max-size": "100m",
            "max-file": "3",
            "compress": "true"
          }' >"${candidate}"
  fi

  if ! dockerd --validate --config-file "${candidate}" >/dev/null; then
    rm -f "${candidate}"
    die "the merged Docker daemon configuration is invalid"
  fi
  if [[ -e "${DAEMON_CONFIG}" ]] && cmp -s "${candidate}" "${DAEMON_CONFIG}"; then
    rm -f "${candidate}"
    log "Docker logging configuration is already current."
    return
  fi

  protect_manager_quorum
  install -m 0644 "${candidate}" "${DAEMON_CONFIG}"
  rm -f "${candidate}"
  docker_config_changed=true
  log "Updated ${DAEMON_CONFIG} with bounded json-file log rotation."
}

if [[ -z "${installed_docker}" || ( -n "${docker_version}" && "${installed_docker}" != "${docker_version}" ) ]]; then
  install_docker
fi

configure_docker_logging
run systemctl enable docker
if [[ "${docker_config_changed}" == true ]]; then
  run systemctl restart docker
else
  run systemctl start docker
fi
if [[ "${dry_run}" != true ]]; then
  docker info >/dev/null
  actual_docker=$(docker version --format '{{.Server.Version}}')
  [[ -z "${docker_version}" || "${actual_docker}" == "${docker_version}" ]] ||
    die "Docker version verification failed: expected ${docker_version}, found ${actual_docker}"
  log "Verified Docker Engine ${actual_docker}; Nectar will record it as the cluster-wide target version."
else
  log "Would record the verified Docker Engine version as the cluster-wide target after Docker starts."
fi

run docker image pull "${image}"

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
  docker node update --label-add nectar.control=true "${manager_node_id}" >/dev/null
else
  log "Would label the current Manager node nectar.control=true."
fi

service_exists=false
if [[ "${dry_run}" != true ]] && docker service inspect "${STACK_NAME}_nectar" >/dev/null 2>&1; then
  service_exists=true
fi
if [[ "${service_exists}" == true ]]; then
  recorded_service_version=$(
    docker service inspect --format '{{range .Spec.TaskTemplate.ContainerSpec.Env}}{{println .}}{{end}}' "${STACK_NAME}_nectar" |
      awk -F= '$1 == "NECTAR_DESIRED_DOCKER_VERSION" {print substr($0, index($0, "=") + 1); exit}'
  )
  if [[ -n "${recorded_service_version}" && "${recorded_service_version}" != "${actual_docker}" ]]; then
    die "Docker ${actual_docker} is running, but Nectar records ${recorded_service_version} as the cluster target; use a controlled cluster Docker upgrade instead of overwriting the policy"
  fi
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
  nectar:
    image: "${image}"
    user: root
    environment:
      NECTAR_ADDR: ":8080"
      NECTAR_COOKIE_SECURE: "false"
      NECTAR_DATA_DIR: /var/lib/nectar
      NECTAR_DESIRED_DOCKER_VERSION: "${actual_docker}"
      NECTAR_INIT_TOKEN_FILE: /run/secrets/${SECRET_NAME}
      NECTAR_REQUIRE_DOCKER: "true"
    ports:
      - target: 8080
        published: ${web_port}
        protocol: tcp
        mode: ingress
    volumes:
      - nectar_data:/var/lib/nectar
      - /var/run/docker.sock:/var/run/docker.sock
    secrets:
      - ${SECRET_NAME}
    deploy:
      labels:
        io.nectar.managed: "true"
      replicas: 1
      placement:
        constraints:
          - node.role == manager
          - node.labels.nectar.control == true
      restart_policy:
        condition: any
        delay: 5s
        max_attempts: 5
      update_config:
        parallelism: 1
        order: stop-first
        failure_action: rollback
volumes:
  nectar_data:
secrets:
  ${SECRET_NAME}:
    external: true
EOF
  chmod 0644 "${stack_file}"
fi

run docker stack deploy --detach=true "${STACK_NAME}" --compose-file "${stack_file}"

setup_url="http://${advertise_addr}:${web_port}/"
if [[ "${dry_run}" != true ]]; then
  log "Waiting for Nectar readiness at ${setup_url}health/ready …"
  ready=false
  for _ in $(seq 1 60); do
    if curl -fsS --max-time 3 "${setup_url}health/ready" >/dev/null 2>&1; then
      ready=true
      break
    fi
    sleep 2
  done
  [[ "${ready}" == true ]] || die "Nectar did not become ready; inspect: docker service ps ${STACK_NAME}_nectar"

  printf '\nNectar is ready.\n'
  printf 'Setup URL: %s\n' "${setup_url}"
  printf 'One-time setup token: %s\n' "$(<"${token_file}")"
  printf 'The token file is root-readable at %s for safe installer resume. Delete it after setup.\n' "${token_file}"
else
  printf '\nDry run completed. No host changes were made.\n'
  printf 'Planned setup URL: %s\n' "${setup_url}"
fi
