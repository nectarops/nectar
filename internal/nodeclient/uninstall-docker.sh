#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-only

set -Eeuo pipefail
IFS=$'\n\t'

readonly PROGRAM="nectar-docker-uninstaller"

dry_run=false
assume_yes=false
purge_data=false
purge_config=false
purge_containerd_data=false
purge_static_binaries=false
force_swarm_manager=false
force_unavailable_daemon=false
force_shared_containerd=false

package_manager=""
docker_root="/var/lib/docker"
docker_path=""
dockerd_path=""
docker_daemon_available=false
swarm_state="unknown"
swarm_manager=false
swarm_node_id=""
remove_containerd_service=false
snap_docker_installed=false
shared_containerd_detected=false
containerd_package_preserved=false

declare -a package_candidates=()
declare -a installed_packages=()

log() {
  printf '[%s] %s\n' "${PROGRAM}" "$*" >&2
}

die() {
  log "ERROR: $*"
  exit 1
}

usage() {
  cat <<'EOF'
Safely uninstall Docker Engine from a Linux host.

Usage: sudo bash uninstall-docker.sh [options]

Options:
  --dry-run                   Print the planned operations without changing the host.
  --yes                       Skip the interactive confirmation.
  --purge-data                Delete Docker's data root, including images, containers,
                              named volumes, networks, Swarm state, configs, and secrets.
  --purge-config              Delete Docker daemon configuration and official repository files.
  --purge-containerd-data     Also delete /var/lib/containerd. Requires --purge-data.
  --purge-static-binaries     Delete manually installed Docker/runtime binaries under /usr/local.
  --all                       Enable every purge option above.
  --force-swarm-manager       Allow this Manager to leave its Swarm with --force.
                              This can destroy quorum and stop Swarm workloads.
  --force-unavailable-daemon  Continue when Docker cannot be queried but Swarm state exists
                              under the Docker data root.
  --force-shared-containerd   Delete containerd data even when Kubernetes, k3s, or RKE2
                              state is detected on the host.
  --help                      Show this help.

Supported package-manager families:
  apt/dpkg, dnf, yum, zypper/rpm, pacman, and apk. Docker installed as a snap is
  removed when detected. Package-managed shared containerd/runc installations are
  preserved unless the package itself is Docker's containerd.io package.

Examples:
  sudo bash uninstall-docker.sh --dry-run --all
  sudo bash uninstall-docker.sh --all
  sudo bash uninstall-docker.sh --all --yes

For an existing Swarm Manager, inspect quorum and migrate workloads first, then add
--force-swarm-manager explicitly. Bind-mounted host data and per-user rootless Docker
state are never deleted automatically.
EOF
}

run() {
  if [[ "${dry_run}" == true ]]; then
    printf '[dry-run]'
    printf ' %q' "$@"
    printf '\n'
    return 0
  fi
  "$@"
}

run_optional() {
  if [[ "${dry_run}" == true ]]; then
    run "$@"
    return 0
  fi
  if ! "$@"; then
    log "WARNING: optional cleanup command failed: $*"
  fi
}

parse_args() {
  while (($# > 0)); do
    case "$1" in
      --dry-run)
        dry_run=true
        ;;
      --yes)
        assume_yes=true
        ;;
      --purge-data)
        purge_data=true
        ;;
      --purge-config)
        purge_config=true
        ;;
      --purge-containerd-data)
        purge_containerd_data=true
        ;;
      --purge-static-binaries)
        purge_static_binaries=true
        ;;
      --all)
        purge_data=true
        purge_config=true
        purge_containerd_data=true
        purge_static_binaries=true
        ;;
      --force-swarm-manager)
        force_swarm_manager=true
        ;;
      --force-unavailable-daemon)
        force_unavailable_daemon=true
        ;;
      --force-shared-containerd)
        force_shared_containerd=true
        ;;
      --help | -h)
        usage
        exit 0
        ;;
      *) die "unknown argument: $1" ;;
    esac
    shift
  done

  if [[ "${purge_containerd_data}" == true && "${purge_data}" != true ]]; then
    die "--purge-containerd-data requires --purge-data"
  fi
}

require_linux_root() {
  [[ "$(uname -s)" == "Linux" ]] || die "this script supports Linux hosts only"
  [[ "${EUID}" -eq 0 ]] || die "run this script as root (for example, with sudo)"
}

configure_local_docker_endpoint() {
  unset DOCKER_CONTEXT DOCKER_TLS DOCKER_TLS_VERIFY DOCKER_CERT_PATH
  export DOCKER_HOST="unix:///var/run/docker.sock"
}

detect_package_manager() {
  if command -v apt-get >/dev/null 2>&1 && command -v dpkg-query >/dev/null 2>&1; then
    package_manager="apt"
  elif command -v dnf >/dev/null 2>&1 && command -v rpm >/dev/null 2>&1; then
    package_manager="dnf"
  elif command -v yum >/dev/null 2>&1 && command -v rpm >/dev/null 2>&1; then
    package_manager="yum"
  elif command -v zypper >/dev/null 2>&1 && command -v rpm >/dev/null 2>&1; then
    package_manager="zypper"
  elif command -v pacman >/dev/null 2>&1; then
    package_manager="pacman"
  elif command -v apk >/dev/null 2>&1; then
    package_manager="apk"
  else
    die "no supported package manager found; use the vendor's uninstall procedure for this distribution"
  fi
}

configure_package_candidates() {
  case "${package_manager}" in
    apt)
      package_candidates=(
        docker-ce
        docker-ce-cli
        docker-ce-rootless-extras
        docker-buildx-plugin
        docker-compose-plugin
        docker-scan-plugin
        docker-model-plugin
        docker-secrets-engine
        docker-secrets-engine-plugins
        containerd.io
        docker.io
        docker-compose-v2
        docker-compose
        moby-engine
        moby-cli
        moby-buildx
        moby-compose
        moby-containerd
        moby-runc
      )
      ;;
    dnf | yum | zypper)
      package_candidates=(
        docker-ce
        docker-ce-cli
        docker-ce-rootless-extras
        docker-buildx-plugin
        docker-compose-plugin
        docker-scan-plugin
        docker-model-plugin
        docker-secrets-engine
        docker-secrets-engine-plugins
        containerd.io
        docker
        docker-client
        docker-client-latest
        docker-common
        docker-latest
        docker-latest-logrotate
        docker-logrotate
        docker-selinux
        docker-engine-selinux
        docker-engine
        moby-engine
        moby-cli
        moby-buildx
        moby-compose
        moby-containerd
        moby-runc
      )
      ;;
    pacman)
      package_candidates=(
        docker
        docker-buildx
        docker-compose
        docker-rootless-extras
        moby-engine
        moby-cli
      )
      ;;
    apk)
      package_candidates=(
        docker
        docker-cli
        docker-cli-buildx
        docker-cli-compose
        docker-engine
        docker-openrc
        docker-rootless-extras
      )
      ;;
    *) die "unsupported package manager: ${package_manager}" ;;
  esac
}

package_installed() {
  local package=$1

  case "${package_manager}" in
    apt)
      # dpkg-query expands this format expression; the shell must keep it literal.
      # shellcheck disable=SC2016
      [[ "$(dpkg-query -W -f='${db:Status-Abbrev}' "${package}" 2>/dev/null || true)" == "ii " ]]
      ;;
    dnf | yum | zypper)
      rpm -q "${package}" >/dev/null 2>&1
      ;;
    pacman)
      pacman -Q "${package}" >/dev/null 2>&1
      ;;
    apk)
      apk info -e "${package}" >/dev/null 2>&1
      ;;
    *) return 1 ;;
  esac
}

discover_installed_packages() {
  local package

  if [[ -e /etc/kubernetes || -e /var/lib/kubelet || -e /var/lib/rancher/k3s || -e /var/lib/rancher/rke2 ]]; then
    shared_containerd_detected=true
  fi

  installed_packages=()
  for package in "${package_candidates[@]}"; do
    if package_installed "${package}"; then
      if [[ "${package}" == "containerd.io" && "${shared_containerd_detected}" == true &&
        "${force_shared_containerd}" != true ]]; then
        containerd_package_preserved=true
        continue
      fi
      installed_packages+=("${package}")
    fi
  done

  for package in "${installed_packages[@]}"; do
    if [[ "${package}" == "containerd.io" ]]; then
      remove_containerd_service=true
      break
    fi
  done

  if command -v snap >/dev/null 2>&1 && snap list docker >/dev/null 2>&1; then
    snap_docker_installed=true
  fi

  if [[ "${purge_containerd_data}" == true ]]; then
    remove_containerd_service=true
  fi
}

validate_data_root() {
  local path=$1

  [[ "${path}" == /* ]] || die "Docker data root must be an absolute path: ${path}"
  [[ "${path}" != *$'\n'* && "${path}" != *$'\r'* ]] || die "Docker data root contains a line break"
  case "${path}" in
    / | /bin | /boot | /dev | /etc | /home | /lib | /lib64 | /mnt | /opt | /proc | /root | /run | /sbin | /srv | /sys | /tmp | /usr | /var | /var/lib)
      die "refusing unsafe Docker data root: ${path}"
      ;;
  esac
  ((${#path} >= 8)) || die "refusing suspiciously broad Docker data root: ${path}"
}

inspect_docker() {
  local detected_root=""

  docker_path=$(command -v docker 2>/dev/null || true)
  dockerd_path=$(command -v dockerd 2>/dev/null || true)
  if [[ -n "${docker_path}" ]] && docker info >/dev/null 2>&1; then
    docker_daemon_available=true
    detected_root=$(docker info --format '{{.DockerRootDir}}' 2>/dev/null || true)
    [[ -z "${detected_root}" ]] || docker_root=${detected_root}
    swarm_state=$(docker info --format '{{.Swarm.LocalNodeState}}' 2>/dev/null || true)
    swarm_node_id=$(docker info --format '{{.Swarm.NodeID}}' 2>/dev/null || true)
    if [[ "$(docker info --format '{{.Swarm.ControlAvailable}}' 2>/dev/null || true)" == "true" ]]; then
      swarm_manager=true
    fi
  fi
  validate_data_root "${docker_root}"
}

validate_safety() {
  if [[ "${swarm_state}" == "active" && "${swarm_manager}" == true && "${force_swarm_manager}" != true ]]; then
    if [[ "${dry_run}" == true ]]; then
      log "WARNING: this host is a Swarm Manager; a real run requires --force-swarm-manager."
    else
      die "this host is a Swarm Manager; demote it safely or pass --force-swarm-manager after reviewing quorum"
    fi
  fi

  if [[ "${docker_daemon_available}" != true && -d "${docker_root}/swarm" && "${force_unavailable_daemon}" != true ]]; then
    if [[ "${dry_run}" == true ]]; then
      log "WARNING: Docker is unavailable but ${docker_root}/swarm exists; a real run requires --force-unavailable-daemon."
    else
      die "Docker is unavailable but Swarm state exists; repair Docker or pass --force-unavailable-daemon"
    fi
  fi

  if [[ "${purge_containerd_data}" == true && "${shared_containerd_detected}" == true && "${force_shared_containerd}" != true ]]; then
    if [[ "${dry_run}" == true ]]; then
      log "WARNING: Kubernetes, k3s, or RKE2 state exists; a real containerd purge requires --force-shared-containerd."
    else
      die "containerd may be shared with Kubernetes, k3s, or RKE2; omit its purge or pass --force-shared-containerd"
    fi
  fi
}

print_plan() {
  local packages="none detected"

  if ((${#installed_packages[@]} > 0)); then
    packages=$(
      IFS=,
      printf '%s' "${installed_packages[*]}"
    )
  fi
  cat >&2 <<EOF
[${PROGRAM}] Uninstall plan
  package manager:          ${package_manager}
  installed Docker packages: ${packages}
  Docker CLI:              ${docker_path:-not found}
  Docker daemon:           ${dockerd_path:-not found}
  daemon reachable:        ${docker_daemon_available}
  Swarm state:             ${swarm_state}
  Swarm Manager:           ${swarm_manager}
  Swarm Node ID:           ${swarm_node_id:-none}
  Docker data root:        ${docker_root}
  purge Docker data:       ${purge_data}
  purge daemon config:     ${purge_config}
  purge containerd data:   ${purge_containerd_data}
  shared containerd use:   ${shared_containerd_detected}
  preserve containerd.io:  ${containerd_package_preserved}
  purge static binaries:   ${purge_static_binaries}

WARNING: removing Docker packages stops every local container. Purging data permanently
deletes images, containers, named volumes, networks, Swarm state, configs, and secrets.
Bind-mounted host directories and per-user rootless Docker data are not removed.
Deleting /var/lib/containerd can also destroy Kubernetes, k3s, or RKE2 workload state.
EOF
}

confirm_uninstall() {
  local confirmation=""

  if [[ "${dry_run}" == true || "${assume_yes}" == true ]]; then
    return 0
  fi
  [[ -t 0 ]] || die "interactive confirmation is unavailable; review --dry-run, then pass --yes"
  printf 'Type DELETE DOCKER to continue: ' >&2
  read -r confirmation
  [[ "${confirmation}" == "DELETE DOCKER" ]] || die "confirmation did not match; no changes were made"
}

leave_swarm() {
  [[ "${swarm_state}" == "active" && "${docker_daemon_available}" == true ]] || return 0

  if [[ "${swarm_manager}" == true ]]; then
    log "Leaving the Swarm as Manager ${swarm_node_id}; quorum and Swarm services may be lost."
    run docker swarm leave --force
    return 0
  fi
  log "Leaving the Swarm as Worker ${swarm_node_id}."
  run docker swarm leave
}

stop_services() {
  local service_manager_found=false

  if [[ -d /run/systemd/system ]] && command -v systemctl >/dev/null 2>&1; then
    service_manager_found=true
    run_optional systemctl disable --now docker.service
    run_optional systemctl disable --now docker.socket
    if [[ "${remove_containerd_service}" == true ]]; then
      run_optional systemctl disable --now containerd.service
    fi
  elif command -v rc-service >/dev/null 2>&1; then
    service_manager_found=true
    run_optional rc-service docker stop
    if command -v rc-update >/dev/null 2>&1; then
      run_optional rc-update del docker default
    fi
    if [[ "${remove_containerd_service}" == true ]]; then
      run_optional rc-service containerd stop
      if command -v rc-update >/dev/null 2>&1; then
        run_optional rc-update del containerd default
      fi
    fi
  elif command -v service >/dev/null 2>&1; then
    service_manager_found=true
    run_optional service docker stop
    if [[ "${remove_containerd_service}" == true ]]; then
      run_optional service containerd stop
    fi
  fi

  if command -v pgrep >/dev/null 2>&1 && pgrep -x dockerd >/dev/null 2>&1 &&
    command -v pkill >/dev/null 2>&1; then
    run_optional pkill -TERM -x dockerd
  elif [[ "${service_manager_found}" != true ]]; then
    log "WARNING: no supported service manager or pkill command was found; verify dockerd is stopped manually."
  fi
}

remove_packages() {
  if ((${#installed_packages[@]} == 0)); then
    log "No known package-managed Docker Engine packages were detected."
  else
    case "${package_manager}" in
      apt)
        run apt-get purge -y "${installed_packages[@]}"
        ;;
      dnf)
        run dnf -y remove "${installed_packages[@]}"
        ;;
      yum)
        run yum -y remove "${installed_packages[@]}"
        ;;
      zypper)
        run zypper --non-interactive remove "${installed_packages[@]}"
        ;;
      pacman)
        run pacman -Rns --noconfirm "${installed_packages[@]}"
        ;;
      apk)
        run apk del "${installed_packages[@]}"
        ;;
      *) die "unsupported package manager: ${package_manager}" ;;
    esac
  fi

  if [[ "${snap_docker_installed}" == true ]]; then
    run snap remove --purge docker
  fi
}

remove_static_binaries() {
  local path
  local runtime_binary=false
  local -a paths=(
    /usr/local/bin/docker
    /usr/local/bin/dockerd
    /usr/local/bin/docker-compose
    /usr/local/bin/docker-init
    /usr/local/bin/docker-proxy
    /usr/local/bin/containerd
    /usr/local/bin/containerd-shim
    /usr/local/bin/containerd-shim-runc-v2
    /usr/local/bin/ctr
    /usr/local/bin/runc
    /usr/local/sbin/docker
    /usr/local/sbin/dockerd
    /usr/local/sbin/containerd
    /usr/local/sbin/ctr
    /usr/local/sbin/runc
  )

  [[ "${purge_static_binaries}" == true ]] || return 0
  for path in "${paths[@]}"; do
    [[ -e "${path}" || -L "${path}" ]] || continue
    runtime_binary=false
    case "${path##*/}" in
      containerd | containerd-shim | containerd-shim-runc-v2 | ctr | runc)
        runtime_binary=true
        ;;
    esac
    if [[ "${runtime_binary}" == true && "${shared_containerd_detected}" == true &&
      "${force_shared_containerd}" != true ]]; then
      log "Preserving shared runtime binary ${path}."
      continue
    fi
    run rm -f -- "${path}"
  done
}

remove_config() {
  local path
  local -a paths=(
    /etc/docker
    /etc/systemd/system/docker.service
    /etc/systemd/system/docker.socket
    /etc/systemd/system/docker.service.d
    /etc/apt/sources.list.d/docker.list
    /etc/apt/sources.list.d/docker.sources
    /etc/apt/keyrings/docker.asc
    /etc/apt/keyrings/docker.gpg
    /usr/share/keyrings/docker-archive-keyring.gpg
    /etc/yum.repos.d/docker-ce.repo
    /etc/zypp/repos.d/docker-ce.repo
  )

  [[ "${purge_config}" == true ]] || return 0
  for path in "${paths[@]}"; do
    [[ -e "${path}" || -L "${path}" ]] || continue
    run rm -rf -- "${path}"
  done
  if command -v systemctl >/dev/null 2>&1; then
    run_optional systemctl daemon-reload
  fi
}

remove_data() {
  [[ "${purge_data}" == true ]] || return 0
  validate_data_root "${docker_root}"
  if [[ -e "${docker_root}" || -L "${docker_root}" ]]; then
    run rm -rf -- "${docker_root}"
  fi
  if [[ "${purge_containerd_data}" == true && -e /var/lib/containerd ]]; then
    run rm -rf -- /var/lib/containerd
  fi
  if [[ -S /var/run/docker.sock || -e /var/run/docker.sock ]]; then
    run rm -f -- /var/run/docker.sock
  fi
  if [[ -S /run/docker.sock || -e /run/docker.sock ]]; then
    run rm -f -- /run/docker.sock
  fi
}

report_remaining_manual_install() {
  local remaining_dockerd=""

  remaining_dockerd=$(command -v dockerd 2>/dev/null || true)
  if [[ -n "${remaining_dockerd}" && "${dry_run}" != true ]]; then
    log "WARNING: dockerd remains at ${remaining_dockerd}; it may belong to an unrecognized or manual installation."
  fi
  if [[ -d /root/.local/share/docker || -d /root/.config/docker ]]; then
    log "WARNING: rootless/per-user Docker state was preserved; remove it as the owning user after inspection."
  fi
}

main() {
  parse_args "$@"
  require_linux_root
  configure_local_docker_endpoint
  detect_package_manager
  configure_package_candidates
  discover_installed_packages
  inspect_docker
  validate_safety
  print_plan
  confirm_uninstall

  leave_swarm
  stop_services
  remove_packages
  remove_static_binaries
  remove_config
  remove_data
  report_remaining_manual_install

  if [[ "${dry_run}" == true ]]; then
    log "Dry run completed. No host changes were made."
  else
    log "Docker Engine uninstall completed. Reboot if stale mounts or network interfaces remain."
  fi
}

if [[ "${NECTAR_UNINSTALL_SOURCE_ONLY:-false}" != true ]]; then
  main "$@"
fi
