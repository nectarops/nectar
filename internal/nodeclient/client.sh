#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-only

set -Eeuo pipefail
IFS=$'\n\t'

readonly PROGRAM="nectar-node-client"
readonly DOCKER_DEB_KEY_FINGERPRINT="9DC858229FC7DD38854AE2D88D81803C0EBFCD88"
readonly DOCKER_RPM_KEY_FINGERPRINT="060A61C51B558A7F742B77AAC52FEB6B621E9F35"
readonly DEFAULT_DOCKER_REPOSITORY_URL="https://download.docker.com/linux"

enrollment_token=""
machine_id_hash=""
auth_config=""
bootstrap_file=""
complete_file=""
join_request_file=""
join_response_file=""
work_dir=""
failure_reported=false

log() {
  printf '[%s] %s\n' "${PROGRAM}" "$*" >&2
}

cleanup() {
  [[ -z "${auth_config}" ]] || rm -f -- "${auth_config}"
  [[ -z "${bootstrap_file}" ]] || rm -f -- "${bootstrap_file}"
  [[ -z "${complete_file}" ]] || rm -f -- "${complete_file}"
  [[ -z "${join_request_file}" ]] || rm -f -- "${join_request_file}"
  [[ -z "${join_response_file}" ]] || rm -f -- "${join_response_file}"
  if [[ -n "${work_dir}" ]]; then
    rmdir -- "${work_dir}" 2>/dev/null || true
  fi
}

installed_docker_version() {
  if ! command -v docker >/dev/null 2>&1; then
    return 0
  fi
  docker version --format '{{.Server.Version}}' 2>/dev/null || true
}

report_progress() {
  local phase=$1
  local docker_version=${2:-}

  [[ -n "${auth_config}" && -n "${machine_id_hash}" ]] || return 0
  curl --config "${auth_config}" \
    --connect-timeout 15 \
    --max-time 30 \
    --retry 2 \
    --retry-all-errors \
    --silent \
    --show-error \
    --fail-with-body \
    --request POST \
    --data-urlencode "machineIdHash=${machine_id_hash}" \
    --data-urlencode "phase=${phase}" \
    --data-urlencode "dockerVersion=${docker_version}" \
    "${NECTAR_SERVER_URL}/api/v1/node-enrollments/progress" \
    >/dev/null
}

die() {
  local message=$1
  if [[ "${failure_reported}" != true ]]; then
    failure_reported=true
    report_progress failed "$(installed_docker_version)" >/dev/null 2>&1 || true
  fi
  log "ERROR: ${message}"
  log "Fix the reported condition, then rerun the same enrollment command before it expires."
  exit 1
}

on_error() {
  local exit_code=$?
  local line=${BASH_LINENO[0]:-unknown}
  trap - ERR
  set +e
  if [[ "${failure_reported}" != true ]]; then
    failure_reported=true
    report_progress failed "$(installed_docker_version)" >/dev/null 2>&1
  fi
  log "Enrollment stopped at line ${line} (exit ${exit_code}). No token value was logged."
  exit "${exit_code}"
}

trap cleanup EXIT
trap on_error ERR

validate_server_url() {
  [[ "${NECTAR_SERVER_URL}" =~ ^https?://[A-Za-z0-9._:-]+$ ]] ||
    die "NECTAR_SERVER_URL must be an HTTP(S) origin without a path or query"
  NECTAR_SERVER_URL=${NECTAR_SERVER_URL%/}
  if [[ "${NECTAR_SERVER_URL}" == http://* ]]; then
    log "WARNING: enrollment is using HTTP; restrict it to a trusted private network and enable HTTPS promptly."
  fi
}

detect_platform() {
  [[ -r /etc/os-release ]] || die "/etc/os-release is required"
  # shellcheck disable=SC1091
  source /etc/os-release
  distribution=${ID,,}
  operating_system=${PRETTY_NAME:-${NAME:-${distribution}}}

  case "${distribution}" in
    ubuntu | debian)
      package_family="deb"
      ;;
    centos)
      centos_major=${VERSION_ID%%.*}
      centos_name=${NAME:-}
      centos_name=${centos_name,,}
      [[ "${centos_name}" == "centos stream" ]] ||
        die "supported CentOS releases are CentOS Stream 9 and 10"
      case "${centos_major}" in
        9 | 10) ;;
        *) die "supported CentOS releases are CentOS Stream 9 and 10" ;;
      esac
      command -v dnf >/dev/null 2>&1 || die "dnf is required on CentOS Stream"
      package_family="rpm"
      ;;
    *) die "supported distributions are Ubuntu, Debian, and CentOS Stream 9 or 10" ;;
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
    *) die "supported architectures are amd64 and arm64" ;;
  esac
}

ensure_client_tools() {
  if command -v curl >/dev/null 2>&1 && command -v ip >/dev/null 2>&1 &&
    command -v sha256sum >/dev/null 2>&1; then
    return
  fi

  case "${package_family}" in
    deb)
      apt-get update
      apt-get install -y ca-certificates coreutils curl iproute2
      ;;
    rpm)
      dnf -y install ca-certificates coreutils iproute
      if ! command -v curl >/dev/null 2>&1; then
        dnf -y install curl-minimal
      fi
      ;;
    *) die "unsupported package family: ${package_family}" ;;
  esac
}

resolve_server_ipv4() {
  local authority=${NECTAR_SERVER_URL#*://}
  local host=${authority%%:*}
  local resolved=""

  if [[ "${host}" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    printf '%s\n' "${host}"
    return
  fi
  if command -v getent >/dev/null 2>&1; then
    resolved=$(getent ahostsv4 "${host}" | awk 'NR == 1 {print $1}')
  fi
  [[ -n "${resolved}" ]] || die "unable to resolve an IPv4 address for the Nectar Manager"
  printf '%s\n' "${resolved}"
}

detect_advertise_address() {
  local server_ip
  local detected=""

  if [[ -n "${NECTAR_ADVERTISE_ADDR:-}" ]]; then
    detected=${NECTAR_ADVERTISE_ADDR}
  else
    server_ip=$(resolve_server_ipv4)
    detected=$(ip -4 route get "${server_ip}" 2>/dev/null |
      awk '{for (field = 1; field <= NF; field++) if ($field == "src") {print $(field + 1); exit}}')
  fi
  [[ "${detected}" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]] ||
    die "unable to select an IPv4 advertise address; set NECTAR_ADVERTISE_ADDR"
  printf '%s\n' "${detected}"
}

prepare_authenticated_curl() {
  work_dir=$(mktemp -d /tmp/nectar-node-client.XXXXXX)
  chmod 0700 "${work_dir}"
  auth_config="${work_dir}/curl.conf"
  bootstrap_file="${work_dir}/bootstrap"
  complete_file="${work_dir}/complete"
  join_request_file="${work_dir}/swarm-join.json"
  join_response_file="${work_dir}/swarm-join-response"
  umask 077
  printf 'header = "Authorization: Bearer %s"\n' "${enrollment_token}" >"${auth_config}"
}

bootstrap_value() {
  local key=$1
  awk -F= -v key="${key}" '$1 == key {print substr($0, length(key) + 2); exit}' "${bootstrap_file}"
}

claim_enrollment() {
  local docker_version=$1

  curl --config "${auth_config}" \
    --connect-timeout 15 \
    --max-time 60 \
    --retry 2 \
    --retry-all-errors \
    --silent \
    --show-error \
    --fail-with-body \
    --request POST \
    --data-urlencode "hostname=${hostname}" \
    --data-urlencode "machineIdHash=${machine_id_hash}" \
    --data-urlencode "operatingSystem=${operating_system}" \
    --data-urlencode "architecture=${architecture}" \
    --data-urlencode "advertiseAddress=${advertise_address}" \
    --data-urlencode "dataPathAddress=${data_path_address}" \
    --data-urlencode "dockerVersion=${docker_version}" \
    --output "${bootstrap_file}" \
    "${NECTAR_SERVER_URL}/api/v1/node-enrollments/claim"
}

validate_bootstrap() {
  enrollment_id=$(bootstrap_value ENROLLMENT_ID)
  requested_role=$(bootstrap_value REQUESTED_ROLE)
  docker_target_version=$(bootstrap_value DOCKER_TARGET_VERSION)
  manager_address=$(bootstrap_value MANAGER_ADDRESS)
  swarm_cluster_id=$(bootstrap_value SWARM_CLUSTER_ID)
  worker_join_token=$(bootstrap_value WORKER_JOIN_TOKEN)

  [[ "${enrollment_id}" =~ ^ne_[A-Za-z0-9_-]{20,80}$ ]] || die "Manager returned an invalid enrollment ID"
  [[ "${requested_role}" == "worker" || "${requested_role}" == "manager" ]] ||
    die "Manager returned an invalid requested role"
  [[ "${docker_target_version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]] ||
    die "Manager returned an invalid Docker target version"
  [[ "${manager_address}" =~ ^[A-Za-z0-9_.:-]+:[0-9]{1,5}$ ]] ||
    die "Manager returned an invalid Swarm address"
  [[ "${swarm_cluster_id}" =~ ^[A-Za-z0-9_-]{20,80}$ ]] || die "Manager returned an invalid Swarm cluster ID"
  [[ "${worker_join_token}" =~ ^SWMTKN-[A-Za-z0-9_-]{20,200}$ ]] ||
    die "Manager returned an invalid Worker join credential"
}

download_and_verify_docker_key() {
  local destination=$1
  local expected_fingerprint=$2
  local fingerprint
  local gpg_command

  gpg_command=$(command -v gpg || command -v gpg2 || true)
  [[ -n "${gpg_command}" ]] || die "gpg is required to verify Docker's repository signing key"
  if ! curl --retry 5 --retry-all-errors --connect-timeout 15 -fsSL \
    "${docker_repository_url}/${distribution}/gpg" -o "${destination}.tmp"; then
    rm -f -- "${destination}.tmp"
    die "unable to download Docker's signing key from ${docker_repository_url}"
  fi
  if ! fingerprint=$("${gpg_command}" --show-keys --with-colons "${destination}.tmp" |
    awk -F: '$1 == "fpr" {print $10; exit}'); then
    rm -f -- "${destination}.tmp"
    die "unable to inspect Docker's repository signing key"
  fi
  if [[ "${fingerprint}" != "${expected_fingerprint}" ]]; then
    rm -f -- "${destination}.tmp"
    die "Docker repository signing-key fingerprint did not match"
  fi
  install -m 0644 "${destination}.tmp" "${destination}"
  rm -f -- "${destination}.tmp"
}

install_docker_deb() {
  local codename=${VERSION_CODENAME:-}
  local keyring="/etc/apt/keyrings/docker.asc"
  local package_version=""
  local repo_arch

  repo_arch=$(dpkg --print-architecture)
  [[ "${repo_arch}" == "${architecture}" ]] || die "dpkg architecture does not match the host"
  [[ -n "${codename}" ]] || die "VERSION_CODENAME is missing from /etc/os-release"

  apt-get update
  apt-get install -y ca-certificates curl gpg
  install -m 0755 -d /etc/apt/keyrings
  download_and_verify_docker_key "${keyring}" "${DOCKER_DEB_KEY_FINGERPRINT}"
  printf 'deb [arch=%s signed-by=%s] %s/%s %s stable\n' \
    "${repo_arch}" "${keyring}" "${docker_repository_url}" "${distribution}" "${codename}" \
    >/etc/apt/sources.list.d/docker.list
  apt-get update
  package_version=$(apt-cache madison docker-ce |
    awk -v requested="${docker_target_version}" \
      '$3 ~ ("^5:" requested "-") && !selected {print $3; selected = 1}')
  [[ -n "${package_version}" ]] || die "Docker ${docker_target_version} is unavailable for this distribution"
  apt-get install -y \
    "docker-ce=${package_version}" \
    "docker-ce-cli=${package_version}" \
    containerd.io \
    docker-buildx-plugin \
    docker-compose-plugin
  apt-mark hold docker-ce docker-ce-cli
}

dnf_with_docker_unpinned() {
  local dnf_version
  dnf_version=$(dnf --version 2>/dev/null | sed -n '1p')
  if [[ "${dnf_version}" == dnf5* || "${dnf_version}" == 5.* ]]; then
    dnf -y --setopt=disable_excludes=docker-ce-stable "$@"
  else
    dnf -y --disableexcludes=docker-ce-stable "$@"
  fi
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
  [[ "${repo_arch}" == "${rpm_architecture}" ]] || die "RPM architecture does not match the host"
  dnf -y install ca-certificates gnupg2
  if ! command -v curl >/dev/null 2>&1; then
    dnf -y install curl-minimal
  fi
  install -m 0755 -d /etc/pki/rpm-gpg /etc/yum.repos.d
  download_and_verify_docker_key "${keyring}" "${DOCKER_RPM_KEY_FINGERPRINT}"
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
  dnf -y --refresh makecache

  engine_package_version=$(
    LC_ALL=C dnf_with_docker_unpinned --showduplicates list docker-ce 2>/dev/null |
      awk -v requested="${docker_target_version}" \
        '$1 ~ /^docker-ce\./ && $2 ~ ("^[0-9]+:" requested "-") {print $2}' |
      sort -Vr |
      sed -n '1p'
  )
  [[ -n "${engine_package_version}" ]] ||
    die "Docker ${docker_target_version} is unavailable for this distribution"
  cli_package_version=$(
    LC_ALL=C dnf_with_docker_unpinned --showduplicates list docker-ce-cli 2>/dev/null |
      awk -v requested="${docker_target_version}" \
        '$1 ~ /^docker-ce-cli\./ && $2 ~ ("^[0-9]+:" requested "-") {print $2}' |
      sort -Vr |
      sed -n '1p'
  )
  [[ -n "${cli_package_version}" ]] || die "Docker CLI ${docker_target_version} is unavailable"
  dnf_with_docker_unpinned install \
    "docker-ce-${engine_package_version}" \
    "docker-ce-cli-${cli_package_version}" \
    containerd.io \
    docker-buildx-plugin \
    docker-compose-plugin
}

install_docker() {
  local actual_version

  report_progress installing
  log "Docker is not installed; installing cluster target version ${docker_target_version}."
  case "${package_family}" in
    deb) install_docker_deb ;;
    rpm) install_docker_rpm ;;
    *) die "unsupported package family: ${package_family}" ;;
  esac
  systemctl enable --now docker
  actual_version=$(installed_docker_version)
  [[ "${actual_version}" == "${docker_target_version}" ]] ||
    die "Docker version verification failed: expected ${docker_target_version}, found ${actual_version:-none}"
}

verify_local_swarm() {
  local actual_cluster_id
  actual_cluster_id=$(docker info --format '{{.Swarm.Cluster.ID}}' 2>/dev/null || true)
  [[ "${actual_cluster_id}" == "${swarm_cluster_id}" ]] ||
    die "this host already belongs to a different Docker Swarm"
}

join_swarm() {
  local docker_api_version
  local local_state

  local_state=$(docker info --format '{{.Swarm.LocalNodeState}}')
  case "${local_state}" in
    inactive)
      report_progress joining "${actual_docker_version}"
      log "Joining the Docker Swarm as Worker; the Manager will apply the requested final role."
      docker_api_version=$(docker version --format '{{.Server.APIVersion}}')
      [[ "${docker_api_version}" =~ ^[0-9]+\.[0-9]+$ ]] || die "Docker returned an invalid API version"
      printf '{"ListenAddr":"0.0.0.0:2377","AdvertiseAddr":"%s","DataPathAddr":"%s","RemoteAddrs":["%s"],"JoinToken":"%s","Availability":"active"}\n' \
        "${advertise_address}" \
        "${data_path_address}" \
        "${manager_address}" \
        "${worker_join_token}" \
        >"${join_request_file}"
      curl --unix-socket /var/run/docker.sock \
        --connect-timeout 15 \
        --max-time 60 \
        --silent \
        --show-error \
        --fail-with-body \
        --header 'Content-Type: application/json' \
        --data-binary "@${join_request_file}" \
        --output "${join_response_file}" \
        "http://localhost/v${docker_api_version}/swarm/join"
      verify_local_swarm
      ;;
    active)
      verify_local_swarm
      log "This host is already a member of the target Swarm; reusing its membership."
      ;;
    *) die "Docker Swarm state ${local_state} is not safe for automatic enrollment" ;;
  esac
}

complete_enrollment() {
  local node_id
  node_id=$(docker info --format '{{.Swarm.NodeID}}')
  [[ "${node_id}" =~ ^[A-Za-z0-9_-]{20,80}$ ]] || die "Docker did not return a valid Swarm Node ID"
  report_progress verifying "${actual_docker_version}"
  curl --config "${auth_config}" \
    --connect-timeout 15 \
    --max-time 60 \
    --retry 5 \
    --retry-delay 2 \
    --retry-all-errors \
    --silent \
    --show-error \
    --fail-with-body \
    --request POST \
    --data-urlencode "machineIdHash=${machine_id_hash}" \
    --data-urlencode "nodeId=${node_id}" \
    --output "${complete_file}" \
    "${NECTAR_SERVER_URL}/api/v1/node-enrollments/complete"
  log "Enrollment completed. Requested role: ${requested_role}; Docker Engine: ${actual_docker_version}."
}

main() {
  (($# == 1)) || die "expected exactly one enrollment credential argument"
  [[ "${EUID}" -eq 0 ]] || die "run this command as root (for example, with sudo)"
  enrollment_token=$1
  [[ "${enrollment_token}" =~ ^[A-Za-z0-9_-]{32,128}$ ]] || die "the enrollment credential is invalid"
  : "${NECTAR_SERVER_URL:?NECTAR_SERVER_URL is required}"

  validate_server_url
  detect_platform
  ensure_client_tools
  [[ -r /etc/machine-id ]] || die "/etc/machine-id is required"
  machine_id_hash=$(sha256sum /etc/machine-id | awk '{print $1}')
  hostname=$(hostname)
  advertise_address=$(detect_advertise_address)
  data_path_address=${NECTAR_DATA_PATH_ADDR:-${advertise_address}}
  [[ "${data_path_address}" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]] ||
    die "NECTAR_DATA_PATH_ADDR must be an IPv4 address"

  docker_repository_url=${NECTAR_DOCKER_REPOSITORY_URL:-${DEFAULT_DOCKER_REPOSITORY_URL}}
  docker_repository_url=${docker_repository_url%/}
  [[ "${docker_repository_url}" =~ ^https://[A-Za-z0-9.-]+(:[0-9]+)?(/[A-Za-z0-9._/-]+)*$ ]] ||
    die "Docker repository URL must be HTTPS without query parameters"

  prepare_authenticated_curl
  current_docker_version=$(installed_docker_version)
  if command -v docker >/dev/null 2>&1 && [[ -z "${current_docker_version}" ]]; then
    die "a Docker CLI is installed but its daemon is unavailable; repair it before enrolling"
  fi
  claim_enrollment "${current_docker_version}"
  validate_bootstrap

  if [[ -z "${current_docker_version}" ]]; then
    install_docker
  else
    log "Preserving existing Docker Engine ${current_docker_version}; cluster target is ${docker_target_version}."
  fi
  actual_docker_version=$(installed_docker_version)
  [[ -n "${actual_docker_version}" ]] || die "Docker Engine is unavailable"
  report_progress docker-ready "${actual_docker_version}"
  join_swarm
  complete_enrollment
}

main "$@"
