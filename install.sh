#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-only

set -Eeuo pipefail
IFS=$'\n\t'

readonly PROGRAM="nectar-installer"
readonly DOCKER_DEB_KEY_FINGERPRINT="9DC858229FC7DD38854AE2D88D81803C0EBFCD88"
readonly DOCKER_RPM_KEY_FINGERPRINT="060A61C51B558A7F742B77AAC52FEB6B621E9F35"
readonly DEFAULT_DOCKER_REPOSITORY_URL="https://download.docker.com/linux"
readonly DEFAULT_NECTAR_VERSION="0.1.6"
readonly DEFAULT_WEB_PORT="8080"
readonly INSTALL_DIR="/opt/nectar"
readonly DATA_DIR="/var/lib/nectar"
readonly STACK_NAME="nectar"
readonly SECRET_NAME="nectar_init_token"
readonly NETWORK_NAME="nectar_control"
readonly MANAGEMENT_NETWORK_NAME="traefik-public"
readonly WEB_PUBLISH_MODE="host"
readonly DAEMON_CONFIG="/etc/docker/daemon.json"
readonly SWARM_POOL_MASK_LENGTH="24"

docker_version=""
advertise_addr=""
web_port="${DEFAULT_WEB_PORT}"
nectar_version="${DEFAULT_NECTAR_VERSION}"
network_subnet=${NECTAR_NETWORK_SUBNET:-}
network_subnet_explicit=false
if [[ -n "${network_subnet}" ]]; then
  network_subnet_explicit=true
fi
swarm_address_pool=${NECTAR_SWARM_ADDRESS_POOL:-}
swarm_address_pool_explicit=false
if [[ -n "${swarm_address_pool}" ]]; then
  swarm_address_pool_explicit=true
fi
force_docker_version=false
dry_run=false

docker_config_changed=false
usage() {
  cat <<'EOF'
Install Nectar on an Ubuntu, Debian, or CentOS Stream Docker Swarm Manager.

Usage: sudo bash install.sh [options]

Options:
  --docker-version VERSION       Install or require this Docker Engine version.
  --advertise-addr ADDRESS       Manager advertise address or interface.
  --web-port PORT               Published Web port (default: 8080).
  --nectar-version VERSION       Deploy this pinned Nectar image tag.
  --network-subnet CIDR          Dedicated, unused IPv4 /24 for Nectar's Overlay network.
  --swarm-address-pool CIDR      Default private IPv4 pool for Swarm Overlay networks.
  --force-docker-version        Explicitly allow changing an existing Docker version.
  --dry-run                     Validate and print planned actions without changing the host.
  --help                        Show this help.

Environment:
  NECTAR_IMAGE                  Override the complete pinned container image reference.
  NECTAR_DOCKER_REPOSITORY_URL  Override the Docker CE repository root.
  NECTAR_NETWORK_SUBNET         Override Nectar's automatically selected Overlay subnet.
  NECTAR_SWARM_ADDRESS_POOL     Override the automatically selected Swarm Overlay address pool.
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

valid_management_domain() {
  local domain=$1

  ((${#domain} <= 253)) || return 1
  [[ "${domain}" =~ ^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)+$ ]]
}

ipv4_to_integer() {
  local address=$1
  local first
  local second
  local third
  local fourth
  local octet

  [[ "${address}" =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]] || return 1
  local IFS=.
  read -r first second third fourth <<<"${address}"
  for octet in "${first}" "${second}" "${third}" "${fourth}"; do
    [[ "${octet}" == "0" || "${octet}" != 0* ]] || return 1
    ((10#${octet} <= 255)) || return 1
  done
  printf '%u\n' "$(((10#${first} << 24) | (10#${second} << 16) | (10#${third} << 8) | 10#${fourth}))"
}

cidr_bounds() {
  local cidr=$1
  local address=${cidr%/*}
  local prefix=${cidr#*/}
  local address_integer
  local mask
  local network
  local broadcast

  [[ "${cidr}" == */* && "${prefix}" =~ ^([0-9]|[12][0-9]|3[0-2])$ ]] || return 1
  address_integer=$(ipv4_to_integer "${address}") || return 1
  if ((10#${prefix} == 0)); then
    mask=0
  else
    mask=$(((4294967295 << (32 - 10#${prefix})) & 4294967295))
  fi
  network=$((address_integer & mask))
  broadcast=$((network | (4294967295 ^ mask)))
  printf '%u %u\n' "${network}" "${broadcast}"
}

validate_network_subnet() {
  local subnet=$1
  local address=${subnet%/*}
  local prefix=${subnet#*/}
  local address_integer
  local bounds
  local first
  local network
  local second
  local third
  local fourth

  [[ "${subnet}" == */* && "${prefix}" == "24" ]] ||
    die "Nectar network subnet must be an IPv4 /24 CIDR, for example 172.30.255.0/24"
  address_integer=$(ipv4_to_integer "${address}") ||
    die "Nectar network subnet must contain a valid IPv4 address"
  local IFS=.
  read -r first second third fourth <<<"${address}"
  if ! ((10#${first} == 10 || (\
    10#${first} == 172 && 10#${second} >= 16 && 10#${second} <= 31) || (\
    10#${first} == 192 && 10#${second} == 168))); then
    die "Nectar network subnet must use RFC 1918 private address space"
  fi
  bounds=$(cidr_bounds "${subnet}") || die "Nectar network subnet is invalid"
  IFS=' ' read -r network _ <<<"${bounds}"
  ((address_integer == network)) ||
    die "Nectar network subnet must use the network address; expected the final octet to be 0 for a /24"
}

validate_swarm_address_pool() {
  local pool=$1
  local address=${pool%/*}
  local prefix=${pool#*/}
  local address_integer
  local bounds
  local first
  local network
  local second

  [[ "${pool}" == */* && "${prefix}" =~ ^([89]|1[0-9]|2[0-4])$ ]] ||
    die "Swarm address pool must be an IPv4 CIDR between /8 and /24"
  address_integer=$(ipv4_to_integer "${address}") ||
    die "Swarm address pool must contain a valid IPv4 address"
  local IFS=.
  read -r first second _ _ <<<"${address}"
  if ! ((10#${first} == 10 || (\
    10#${first} == 172 && 10#${second} >= 16 && 10#${second} <= 31) || (\
    10#${first} == 192 && 10#${second} == 168))); then
    die "Swarm address pool must use RFC 1918 private address space"
  fi
  bounds=$(cidr_bounds "${pool}") || die "Swarm address pool is invalid"
  IFS=' ' read -r network _ <<<"${bounds}"
  ((address_integer == network)) ||
    die "Swarm address pool must use the CIDR network address"
  ((10#${prefix} <= 10#${SWARM_POOL_MASK_LENGTH})) ||
    die "Swarm address pool prefix /${prefix} cannot be narrower than the /${SWARM_POOL_MASK_LENGTH} Overlay subnet size"
}

cidrs_overlap() {
  local first_bounds
  local first_start
  local first_end
  local second_bounds
  local second_start
  local second_end

  first_bounds=$(cidr_bounds "$1") || return 1
  second_bounds=$(cidr_bounds "$2") || return 1
  local IFS=' '
  read -r first_start first_end <<<"${first_bounds}"
  read -r second_start second_end <<<"${second_bounds}"
  ((first_start <= second_end && second_start <= first_end))
}

collect_used_ipv4_cidrs() {
  local network_id

  if command -v ip >/dev/null 2>&1; then
    ip -o -4 address show 2>/dev/null | awk '$4 ~ /^[0-9.]+\/[0-9]+$/ {print $4}'
    ip -o -4 route show 2>/dev/null |
      awk '{for (field = 1; field <= NF; field++) if ($field ~ /^[0-9.]+\/[0-9]+$/) {print $field; break}}'
  fi
  if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
    while IFS= read -r network_id; do
      docker network inspect --format '{{range .IPAM.Config}}{{println .Subnet}}{{end}}' \
        "${network_id}" 2>/dev/null || true
    done < <(docker network ls --quiet)
  fi
}

find_overlapping_cidr() {
  local candidate=$1
  local used_cidrs=$2
  local used_cidr

  while IFS= read -r used_cidr; do
    [[ -n "${used_cidr}" ]] || continue
    cidr_bounds "${used_cidr}" >/dev/null 2>&1 || continue
    if cidrs_overlap "${candidate}" "${used_cidr}"; then
      printf '%s\n' "${used_cidr}"
      return 0
    fi
  done <<<"${used_cidrs}"
  return 1
}

validate_existing_docker_network_overlaps() {
  [[ "${dry_run}" != true ]] || return 0
  command -v docker >/dev/null 2>&1 || return 0
  docker info >/dev/null 2>&1 || return 0

  local -a records=()
  local first
  local first_cidr
  local first_driver
  local first_name
  local first_scope
  local i
  local j
  local network_id
  local second
  local second_cidr
  local second_driver
  local second_name
  local second_scope

  while IFS= read -r network_id; do
    while IFS= read -r first; do
      [[ -n "${first}" ]] && records+=("${first}")
    done < <(docker network inspect --format \
      '{{range .IPAM.Config}}{{if .Subnet}}{{$.Name}}|{{$.Driver}}|{{$.Scope}}|{{.Subnet}}{{println}}{{end}}{{end}}' \
      "${network_id}" 2>/dev/null || true)
  done < <(docker network ls --quiet)

  for ((i = 0; i < ${#records[@]}; i++)); do
    IFS='|' read -r first_name first_driver first_scope first_cidr <<<"${records[$i]}"
    cidr_bounds "${first_cidr}" >/dev/null 2>&1 || continue
    for ((j = i + 1; j < ${#records[@]}; j++)); do
      IFS='|' read -r second_name second_driver second_scope second_cidr <<<"${records[$j]}"
      cidr_bounds "${second_cidr}" >/dev/null 2>&1 || continue
      if cidrs_overlap "${first_cidr}" "${second_cidr}"; then
        die "Docker network address overlap detected before daemon restart: ${first_name} (${first_driver}/${first_scope}, ${first_cidr}) overlaps ${second_name} (${second_driver}/${second_scope}, ${second_cidr}). Refusing to restart Docker during an upgrade. Migrate or recreate the conflicting networks first; an existing Swarm default address pool cannot be changed in place."
      fi
    done
  done
}

select_swarm_address_pool() {
  local candidate
  local overlap
  local used_cidrs

  used_cidrs=$(collect_used_ipv4_cidrs)
  if [[ "${swarm_address_pool_explicit}" == true ]]; then
    validate_swarm_address_pool "${swarm_address_pool}"
    if overlap=$(find_overlapping_cidr "${swarm_address_pool}" "${used_cidrs}"); then
      die "Swarm address pool ${swarm_address_pool} overlaps existing address space ${overlap}; choose another pool with --swarm-address-pool"
    fi
    return
  fi

  for candidate in \
    "172.20.0.0/14" \
    "172.24.0.0/14" \
    "172.28.0.0/14" \
    "10.240.0.0/12" \
    "10.224.0.0/12"; do
    if ! find_overlapping_cidr "${candidate}" "${used_cidrs}" >/dev/null; then
      swarm_address_pool=${candidate}
      return
    fi
  done

  die "unable to find an unused private address pool for Swarm Overlay networks; pass --swarm-address-pool after reviewing host routes and Docker networks"
}

select_network_subnet() {
  local candidate
  local driver
  local existing_subnet
  local managed
  local network_subnets
  local network_prefix
  local overlap
  local scope
  local third_octet
  local used_cidrs

  if [[ "${dry_run}" != true ]] && docker network inspect "${NETWORK_NAME}" >/dev/null 2>&1; then
    driver=$(docker network inspect --format '{{.Driver}}' "${NETWORK_NAME}")
    scope=$(docker network inspect --format '{{.Scope}}' "${NETWORK_NAME}")
    managed=$(docker network inspect --format '{{index .Labels "io.nectar.managed"}}' "${NETWORK_NAME}")
    [[ "${driver}" == "overlay" && "${scope}" == "swarm" ]] ||
      die "Docker network ${NETWORK_NAME} already exists but is not a Swarm Overlay network"
    [[ "${managed}" == "true" ]] ||
      die "Docker network ${NETWORK_NAME} already exists but is not managed by Nectar"
    network_subnets=$(docker network inspect \
      --format '{{range .IPAM.Config}}{{println .Subnet}}{{end}}' "${NETWORK_NAME}" | awk 'NF')
    [[ $(printf '%s\n' "${network_subnets}" | awk 'NF {count++} END {print count+0}') -eq 1 ]] ||
      die "Docker network ${NETWORK_NAME} must have exactly one IPv4 subnet"
    existing_subnet=${network_subnets}
    validate_network_subnet "${existing_subnet}"
    if [[ "${network_subnet_explicit}" == true && "${network_subnet}" != "${existing_subnet}" ]]; then
      die "Docker network ${NETWORK_NAME} already uses ${existing_subnet}; refusing to replace it with ${network_subnet}"
    fi
    network_subnet=${existing_subnet}
    log "Reusing Nectar Overlay network ${NETWORK_NAME} (${network_subnet})."
    return
  fi

  used_cidrs=$(collect_used_ipv4_cidrs)
  if [[ -n "${swarm_address_pool}" ]]; then
    used_cidrs=$(printf '%s\n%s\n' "${used_cidrs}" "${swarm_address_pool}")
  fi
  if [[ "${network_subnet_explicit}" == true ]]; then
    validate_network_subnet "${network_subnet}"
    if overlap=$(find_overlapping_cidr "${network_subnet}" "${used_cidrs}"); then
      die "Nectar network subnet ${network_subnet} overlaps existing address space ${overlap}; choose an unused /24 with --network-subnet"
    fi
    return
  fi

  for network_prefix in "172.31" "192.168" "10.255"; do
    for ((third_octet = 255; third_octet >= 224; third_octet--)); do
      candidate="${network_prefix}.${third_octet}.0/24"
      if ! find_overlapping_cidr "${candidate}" "${used_cidrs}" >/dev/null; then
        network_subnet=${candidate}
        return
      fi
    done
  done

  die "unable to find an unused private /24 for Nectar; pass --network-subnet after reviewing Docker networks and host routes"
}

ensure_nectar_network() {
  select_network_subnet
  if [[ "${dry_run}" == true ]]; then
    log "Would create or reuse dedicated Nectar Overlay network ${NETWORK_NAME} (${network_subnet})."
    return
  fi
  if docker network inspect "${NETWORK_NAME}" >/dev/null 2>&1; then
    return
  fi
  if ! docker network create \
    --driver overlay \
    --label io.nectar.managed=true \
    --subnet "${network_subnet}" \
    "${NETWORK_NAME}" >/dev/null; then
    die "unable to create Nectar Overlay network ${NETWORK_NAME} with subnet ${network_subnet}; inspect existing Docker networks and host routes"
  fi
  log "Created Nectar Overlay network ${NETWORK_NAME} (${network_subnet})."
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
    --network-subnet)
      (($# >= 2)) || die "--network-subnet requires a value"
      network_subnet=$2
      network_subnet_explicit=true
      shift 2
      ;;
    --swarm-address-pool)
      (($# >= 2)) || die "--swarm-address-pool requires a value"
      swarm_address_pool=$2
      swarm_address_pool_explicit=true
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
    --help | -h)
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
if [[ "${network_subnet_explicit}" == true ]]; then
  validate_network_subnet "${network_subnet}"
fi
if [[ "${swarm_address_pool_explicit}" == true ]]; then
  validate_swarm_address_pool "${swarm_address_pool}"
fi

[[ -r /etc/os-release ]] || die "/etc/os-release is required"
# shellcheck disable=SC1091
source /etc/os-release
distribution=${ID,,}
case "${distribution}" in
  ubuntu | debian)
    package_family="deb"
    ;;
  centos)
    centos_major=${VERSION_ID%%.*}
    centos_name=${NAME:-}
    centos_name=${centos_name,,}
    [[ "${centos_name}" == "centos stream" ]] ||
      die "supported CentOS releases are CentOS Stream 9 and 10; found ${PRETTY_NAME:-${NAME:-unknown}}"
    case "${centos_major}" in
      9 | 10) ;;
      *) die "supported CentOS releases are CentOS Stream 9 and 10; found ${PRETTY_NAME:-${VERSION_ID:-unknown}}" ;;
    esac
    command -v dnf >/dev/null 2>&1 || die "dnf is required on CentOS Stream"
    package_family="rpm"
    ;;
  *) die "supported distributions are Ubuntu, Debian, and CentOS Stream 9 or 10; found ${ID:-unknown}" ;;
esac

case "$(uname -m)" in
  x86_64)
    architecture="amd64"
    rpm_architecture="x86_64"
    ;;
  aarch64 | arm64)
    architecture="arm64"
    rpm_architecture="aarch64"
    ;;
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

download_and_verify_docker_key() {
  local destination=$1
  local expected_fingerprint=$2
  local fingerprint
  local gpg_command

  if [[ "${dry_run}" == true ]]; then
    log "Would download and verify Docker's repository signing key fingerprint ${expected_fingerprint}."
    return
  fi

  gpg_command=$(command -v gpg || command -v gpg2 || true)
  [[ -n "${gpg_command}" ]] || die "gpg is required to verify Docker's repository signing key"
  if ! curl --retry 5 --retry-all-errors --connect-timeout 15 -fsSL \
    "${docker_repository_url}/${distribution}/gpg" -o "${destination}.tmp"; then
    rm -f "${destination}.tmp"
    die "unable to download Docker's signing key from ${docker_repository_url}; check outbound HTTPS or set NECTAR_DOCKER_REPOSITORY_URL to a trusted mirror"
  fi
  if ! fingerprint=$("${gpg_command}" --show-keys --with-colons "${destination}.tmp" |
    awk -F: '$1 == "fpr" {print $10; exit}'); then
    rm -f "${destination}.tmp"
    die "unable to inspect Docker's repository signing key"
  fi
  [[ "${fingerprint}" == "${expected_fingerprint}" ]] || {
    rm -f "${destination}.tmp"
    die "Docker repository signing-key fingerprint did not match"
  }
  install -m 0644 "${destination}.tmp" "${destination}"
  rm -f "${destination}.tmp"
}

install_docker_deb() {
  local keyring="/etc/apt/keyrings/docker.asc"
  local repo_arch
  local codename
  local package_version=""

  repo_arch=$(dpkg --print-architecture)
  [[ "${repo_arch}" == "${architecture}" ]] || die "dpkg architecture ${repo_arch} does not match host ${architecture}"
  codename=${VERSION_CODENAME:-}
  [[ -n "${codename}" ]] || die "VERSION_CODENAME is missing from /etc/os-release"

  run apt-get update
  run apt-get install -y ca-certificates curl gpg
  run install -m 0755 -d /etc/apt/keyrings

  download_and_verify_docker_key "${keyring}" "${DOCKER_DEB_KEY_FINGERPRINT}"

  if [[ "${dry_run}" == true ]]; then
    log "Would configure Docker's signed ${distribution} repository for ${codename}."
  else
    printf 'deb [arch=%s signed-by=%s] %s/%s %s stable\n' \
      "${repo_arch}" "${keyring}" "${docker_repository_url}" "${distribution}" "${codename}" >/etc/apt/sources.list.d/docker.list
  fi
  run apt-get update

  if [[ "${dry_run}" != true ]]; then
    if [[ -n "${docker_version}" ]]; then
      package_version=$(apt-cache madison docker-ce |
        awk -v requested="${docker_version}" \
          '$3 ~ ("^5:" requested "-") && !selected {print $3; selected = 1}')
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

dnf_with_docker_unpinned() {
  local dnf_version
  dnf_version=$(dnf --version 2>/dev/null | sed -n '1p')
  if [[ "${dnf_version}" == dnf5* || "${dnf_version}" == 5.* ]]; then
    run dnf -y --setopt=disable_excludes=docker-ce-stable "$@"
  else
    run dnf -y --disableexcludes=docker-ce-stable "$@"
  fi
}

version_is_less() {
  local first
  first=$(printf '%s\n%s\n' "$1" "$2" | sort -V | sed -n '1p')
  [[ "$1" != "$2" && "${first}" == "$1" ]]
}

install_docker_rpm() {
  local cli_package_version=""
  local dnf_basearch="\$basearch"
  local dnf_releasever="\$releasever"
  local engine_package_version=""
  local keyring="/etc/pki/rpm-gpg/docker-ce.asc"
  local repo_arch
  local repo_file="/etc/yum.repos.d/docker-ce.repo"

  repo_arch=$(rpm --eval '%{_arch}')
  [[ "${repo_arch}" == "${rpm_architecture}" ]] ||
    die "RPM architecture ${repo_arch} does not match host ${rpm_architecture}"
  run dnf -y install ca-certificates gnupg2
  if ! command -v curl >/dev/null 2>&1; then
    run dnf -y install curl-minimal
  fi
  run install -m 0755 -d /etc/pki/rpm-gpg /etc/yum.repos.d
  download_and_verify_docker_key "${keyring}" "${DOCKER_RPM_KEY_FINGERPRINT}"

  if [[ "${dry_run}" == true ]]; then
    log "Would configure Docker's signed CentOS Stream repository for release ${centos_major}."
  else
    {
      printf '[docker-ce-stable]\n'
      printf 'name=Docker CE Stable - %s\n' "${dnf_basearch}"
      printf 'baseurl=%s/centos/%s/%s/stable\n' \
        "${docker_repository_url}" "${dnf_releasever}" "${dnf_basearch}"
      printf 'enabled=1\n'
      printf 'gpgcheck=1\n'
      printf 'gpgkey=file://%s\n' "${keyring}"
      printf 'excludepkgs=docker-ce,docker-ce-cli\n'
    } >"${repo_file}"
    chmod 0644 "${repo_file}"
  fi
  run dnf -y --refresh makecache

  if [[ "${dry_run}" == true ]]; then
    log "Would install and pin Docker Engine ${docker_version:-the current repository version}."
    return
  fi

  if [[ -n "${docker_version}" ]]; then
    engine_package_version=$(
      LC_ALL=C dnf_with_docker_unpinned --showduplicates list docker-ce 2>/dev/null |
        awk -v requested="${docker_version}" \
          '$1 ~ /^docker-ce\./ && $2 ~ ("^[0-9]+:" requested "-") {print $2}' |
        sort -Vr |
        sed -n '1p'
    )
    [[ -n "${engine_package_version}" ]] ||
      die "Docker ${docker_version} is not available for this distribution"
  else
    engine_package_version=$(
      LC_ALL=C dnf_with_docker_unpinned --showduplicates list docker-ce 2>/dev/null |
        awk '$1 ~ /^docker-ce\./ && $2 ~ /^[0-9]+:/ {print $2}' |
        sort -Vr |
        sed -n '1p'
    )
    [[ -n "${engine_package_version}" ]] || die "Docker's repository did not provide docker-ce"
    docker_version=${engine_package_version#*:}
    docker_version=${docker_version%%-*}
  fi

  cli_package_version=$(
    LC_ALL=C dnf_with_docker_unpinned --showduplicates list docker-ce-cli 2>/dev/null |
      awk -v requested="${docker_version}" \
        '$1 ~ /^docker-ce-cli\./ && $2 ~ ("^[0-9]+:" requested "-") {print $2}' |
      sort -Vr |
      sed -n '1p'
  )
  [[ -n "${cli_package_version}" ]] ||
    die "Docker CLI ${docker_version} is not available for this distribution"

  if [[ -n "${installed_docker}" ]] && version_is_less "${docker_version}" "${installed_docker}"; then
    dnf_with_docker_unpinned downgrade \
      "docker-ce-${engine_package_version}" \
      "docker-ce-cli-${cli_package_version}"
    dnf_with_docker_unpinned install containerd.io docker-buildx-plugin docker-compose-plugin
  else
    dnf_with_docker_unpinned install \
      "docker-ce-${engine_package_version}" \
      "docker-ce-cli-${cli_package_version}" \
      containerd.io docker-buildx-plugin docker-compose-plugin
  fi
}

install_docker() {
  case "${package_family}" in
    deb) install_docker_deb ;;
    rpm) install_docker_rpm ;;
    *) die "unsupported package family: ${package_family}" ;;
  esac
}

ensure_runtime_tools() {
  if command -v jq >/dev/null 2>&1 && command -v curl >/dev/null 2>&1 && command -v ip >/dev/null 2>&1; then
    return
  fi

  case "${package_family}" in
    deb)
      run apt-get update
      run apt-get install -y ca-certificates curl iproute2 jq
      ;;
    rpm)
      run dnf -y install ca-certificates iproute jq
      if ! command -v curl >/dev/null 2>&1; then
        run dnf -y install curl-minimal
      fi
      ;;
    *) die "unsupported package family: ${package_family}" ;;
  esac
  if [[ "${dry_run}" != true ]]; then
    command -v jq >/dev/null 2>&1 || die "jq is required to merge ${DAEMON_CONFIG} safely"
    command -v curl >/dev/null 2>&1 || die "curl is required for readiness checks"
    command -v ip >/dev/null 2>&1 || die "ip is required to select a non-overlapping Nectar network"
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

  ensure_runtime_tools
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

if [[ -z "${installed_docker}" || (-n "${docker_version}" && "${installed_docker}" != "${docker_version}") ]]; then
  install_docker
fi

# Existing installations can contain local and Swarm networks from independent
# address allocators. Detect overlaps before an installer-triggered Docker restart
# turns a latent conflict into rejected Swarm tasks.
validate_existing_docker_network_overlaps

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
    if [[ "${dry_run}" == true ]]; then
      if [[ -z "${swarm_address_pool}" ]]; then
        swarm_address_pool="172.20.0.0/14"
      fi
      log "Would initialize Swarm with Overlay address pool ${swarm_address_pool} split into /${SWARM_POOL_MASK_LENGTH} networks."
    else
      select_swarm_address_pool
      log "Initializing Swarm with Overlay address pool ${swarm_address_pool} split into /${SWARM_POOL_MASK_LENGTH} networks."
    fi
    run docker swarm init \
      --advertise-addr "${advertise_addr}" \
      --default-addr-pool "${swarm_address_pool}" \
      --default-addr-pool-mask-length "${SWARM_POOL_MASK_LENGTH}"
    ;;
  active)
    if [[ "${dry_run}" != true ]]; then
      control_available=$(docker info --format '{{.Swarm.ControlAvailable}}')
      [[ "${control_available}" == "true" ]] || die "this host already belongs to a Swarm as a Worker; it will not be removed or promoted automatically"
    fi
    log "This host is already an active Swarm Manager; preserving its membership."
    ;;
  pending | locked | error)
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

ensure_nectar_network
log "Nectar Web port ${web_port} will be published in ${WEB_PUBLISH_MODE} mode on the labeled Manager."

service_exists=false
preserved_management_domain=""
if command -v docker >/dev/null 2>&1 &&
  docker service inspect "${STACK_NAME}_nectar" >/dev/null 2>&1; then
  service_exists=true
fi
if [[ "${service_exists}" == true ]]; then
  recorded_service_version=$(
    docker service inspect --format '{{range .Spec.TaskTemplate.ContainerSpec.Env}}{{println .}}{{end}}' "${STACK_NAME}_nectar" |
      awk -F= '$1 == "NECTAR_DESIRED_DOCKER_VERSION" {print substr($0, index($0, "=") + 1); exit}'
  )
  running_docker_version=${actual_docker:-${installed_docker}}
  if [[ -n "${recorded_service_version}" && "${recorded_service_version}" != "${running_docker_version}" ]]; then
    die "Docker ${running_docker_version} is running, but Nectar records ${recorded_service_version} as the cluster target; use a controlled cluster Docker upgrade instead of overwriting the policy"
  fi

  preserved_management_domain=$(docker service inspect \
    --format '{{index .Spec.Labels "io.nectar.management-domain"}}' \
    "${STACK_NAME}_nectar")
  if [[ -n "${preserved_management_domain}" ]]; then
    valid_management_domain "${preserved_management_domain}" ||
      die "the installed Nectar service has an invalid management-domain label; refusing to copy it into the upgraded stack"

    management_network_driver=$(docker network inspect \
      --format '{{.Driver}}' \
      "${MANAGEMENT_NETWORK_NAME}" 2>/dev/null || true)
    management_network_scope=$(docker network inspect \
      --format '{{.Scope}}' \
      "${MANAGEMENT_NETWORK_NAME}" 2>/dev/null || true)
    [[ "${management_network_driver}" == "overlay" && "${management_network_scope}" == "swarm" ]] ||
      die "the installed Nectar service has HTTPS access configured, but ${MANAGEMENT_NETWORK_NAME} is not a Swarm overlay network; repair HTTPS access before upgrading"

    log "Preserving HTTPS management route for ${preserved_management_domain} during the service update."
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
    openssl rand -base64 32 | tr -d '\n' >"${token_file}"
  else
    head -c 32 /dev/urandom | base64 | tr -d '\n' >"${token_file}"
  fi
  chmod 0600 "${token_file}"
fi

if [[ "${dry_run}" != true ]] && ! docker secret inspect "${SECRET_NAME}" >/dev/null 2>&1; then
  docker secret create "${SECRET_NAME}" "${token_file}" >/dev/null
elif [[ "${dry_run}" == true ]]; then
  log "Would create the external ${SECRET_NAME} secret if absent."
fi

stack_file="${INSTALL_DIR}/stack.yml"
write_stack() {
  cat <<EOF
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
        mode: ${WEB_PUBLISH_MODE}
    volumes:
      - nectar_data:/var/lib/nectar
      - /var/run/docker.sock:/var/run/docker.sock
    networks:
      - ${NETWORK_NAME}
EOF
  if [[ -n "${preserved_management_domain}" ]]; then
    cat <<EOF
      - ${MANAGEMENT_NETWORK_NAME}
EOF
  fi
  cat <<EOF
    secrets:
      - ${SECRET_NAME}
    deploy:
      labels:
        io.nectar.managed: "true"
EOF
  if [[ -n "${preserved_management_domain}" ]]; then
    cat <<EOF
        io.nectar.management-domain: "${preserved_management_domain}"
        traefik.enable: "true"
        traefik.swarm.network: "${MANAGEMENT_NETWORK_NAME}"
        traefik.http.routers.nectar.rule: "Host(\`${preserved_management_domain}\`)"
        traefik.http.routers.nectar.entrypoints: "websecure"
        traefik.http.routers.nectar.tls: "true"
        traefik.http.routers.nectar.tls.certresolver: "letsencrypt"
        traefik.http.services.nectar.loadbalancer.server.port: "8080"
EOF
  fi
  cat <<EOF
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
networks:
  ${NETWORK_NAME}:
    external: true
EOF
  if [[ -n "${preserved_management_domain}" ]]; then
    cat <<EOF
  ${MANAGEMENT_NETWORK_NAME}:
    external: true
EOF
  fi
  cat <<EOF
volumes:
  nectar_data:
secrets:
  ${SECRET_NAME}:
    external: true
EOF
}

if [[ "${dry_run}" == true ]]; then
  log "Would write the pinned Swarm stack to ${stack_file}."
else
  write_stack >"${stack_file}"
  chmod 0644 "${stack_file}"
fi

run docker stack deploy --detach=true "${STACK_NAME}" --compose-file "${stack_file}"

setup_url="http://${advertise_addr}:${web_port}/"
if [[ "${dry_run}" != true ]]; then
  readiness_url="http://127.0.0.1:${web_port}/health/ready"
  log "Waiting for local Nectar readiness at ${readiness_url} …"
  ready=false
  latest_task_error=""
  for _ in $(seq 1 60); do
    if curl -fsS --max-time 3 "${readiness_url}" >/dev/null 2>&1; then
      ready=true
      break
    fi
    task_error=$(docker service ps --no-trunc --format '{{.Error}}' "${STACK_NAME}_nectar" 2>/dev/null |
      awk 'NF {print; exit}' || true)
    [[ -z "${task_error}" ]] || latest_task_error=${task_error}
    sleep 2
  done
  if [[ "${ready}" != true ]]; then
    if [[ -n "${latest_task_error}" ]]; then
      die "Nectar did not become ready; latest Swarm task error: ${latest_task_error}; inspect: docker service ps ${STACK_NAME}_nectar --no-trunc"
    fi
    die "Nectar did not become ready; inspect: docker service ps ${STACK_NAME}_nectar --no-trunc"
  fi

  printf '\nNectar is ready.\n'
  printf 'Setup URL: %s\n' "${setup_url}"
  printf 'One-time setup token: %s\n' "$(<"${token_file}")"
  printf 'The token file is root-readable at %s for safe installer resume. Delete it after setup.\n' "${token_file}"
else
  printf '\nDry run completed. No host changes were made.\n'
  printf 'Planned setup URL: %s\n' "${setup_url}"
fi
