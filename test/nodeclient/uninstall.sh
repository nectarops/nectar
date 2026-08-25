#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-only

# This test sources the runtime-selected repository script and mutates globals consumed by
# its functions. ShellCheck cannot follow that dynamic source or connect those assignments.
# shellcheck disable=SC1090,SC1091,SC2034,SC2154,SC2329

set -Eeuo pipefail
IFS=$'\n\t'

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
uninstaller="${repo_root}/internal/nodeclient/uninstall-docker.sh"

fail() {
  printf 'node uninstaller test failed: %s\n' "$*" >&2
  exit 1
}

assert_contains() {
  local output=$1
  local expected=$2

  [[ "${output}" == *"${expected}"* ]] || fail "expected output to contain ${expected}"
}

help_output=$(bash "${uninstaller}" --help)
assert_contains "${help_output}" "--purge-data"
assert_contains "${help_output}" "--force-swarm-manager"
assert_contains "${help_output}" "--force-shared-containerd"
assert_contains "${help_output}" "apt/dpkg, dnf, yum, zypper/rpm, pacman, and apk"

export NECTAR_UNINSTALL_SOURCE_ONLY=true
source "${uninstaller}"

if (validate_data_root / >/dev/null 2>&1); then
  fail "accepted the filesystem root as Docker data root"
fi
if (validate_data_root /var/lib >/dev/null 2>&1); then
  fail "accepted /var/lib as Docker data root"
fi
validate_data_root /var/lib/docker
validate_data_root /mnt/docker-data

export DOCKER_HOST=tcp://remote.example.invalid:2376
export DOCKER_CONTEXT=remote
export DOCKER_TLS_VERIFY=1
configure_local_docker_endpoint
[[ "${DOCKER_HOST}" == "unix:///var/run/docker.sock" ]] || fail "did not force the local Docker socket"
[[ -z "${DOCKER_CONTEXT:-}" ]] || fail "did not clear the remote Docker context"
[[ -z "${DOCKER_TLS_VERIFY:-}" ]] || fail "did not clear remote Docker TLS settings"

purge_data=false
purge_config=false
purge_containerd_data=false
purge_static_binaries=false
parse_args --all
[[ "${purge_data}" == true ]] || fail "--all did not enable Docker data purge"
[[ "${purge_config}" == true ]] || fail "--all did not enable Docker config purge"
[[ "${purge_containerd_data}" == true ]] || fail "--all did not enable containerd data purge"
[[ "${purge_static_binaries}" == true ]] || fail "--all did not enable static binary purge"

package_manager=apt
if ! (
  dpkg-query() {
    printf '%s' 'install ok installed'
  }
  package_installed docker-ce
); then
  fail "APT package detection did not recognize an installed held package"
fi

package_candidates=(docker-ce docker-ce-cli containerd.io docker.io)
purge_containerd_data=false
shared_containerd_detected=false
force_shared_containerd=false
package_installed() {
  case "$1" in
    docker-ce | containerd.io) return 0 ;;
    *) return 1 ;;
  esac
}
discover_installed_packages
[[ "${installed_packages[*]}" == $'docker-ce\ncontainerd.io' ]] ||
  fail "package discovery did not preserve the expected package list"
[[ "${remove_containerd_service}" == true ]] ||
  fail "containerd.io did not enable containerd service cleanup"

apt-mark() {
  [[ "$1" == "showhold" ]] || return 1
  printf '%s\n' docker-ce
}
discover_installed_packages
held_package_plan=$(print_plan 2>&1)
unset -f apt-mark
assert_contains "${held_package_plan}" "held Docker packages:     docker-ce"

remove_containerd_service=false
shared_containerd_detected=true
containerd_package_preserved=false
discover_installed_packages
[[ "${installed_packages[*]}" == "docker-ce" ]] ||
  fail "shared containerd.io package was not preserved"
[[ "${containerd_package_preserved}" == true ]] ||
  fail "shared containerd.io preservation was not reported"
[[ "${remove_containerd_service}" == false ]] ||
  fail "shared containerd service was selected for shutdown"

dry_run=false
swarm_state=active
swarm_manager=true
force_swarm_manager=false
if (validate_safety >/dev/null 2>&1); then
  fail "allowed a real Swarm Manager uninstall without explicit force"
fi

dry_run=true
manager_warning=$(validate_safety 2>&1)
assert_contains "${manager_warning}" "a real run requires --force-swarm-manager"

swarm_manager=false
docker_daemon_available=true
swarm_node_id=worker-node-id
worker_leave_plan=$(leave_swarm)
assert_contains "${worker_leave_plan}" "docker swarm leave"
if [[ "${worker_leave_plan}" == *"--force"* ]]; then
  fail "worker leave plan unexpectedly used --force"
fi

swarm_manager=true
manager_leave_plan=$(leave_swarm)
assert_contains "${manager_leave_plan}" "docker swarm leave --force"

package_manager=apt
installed_packages=(docker-ce docker-ce-cli)
preflight_output=$(
  apt-get() {
    return 0
  }
  preflight_package_removal 2>&1
)
assert_contains "${preflight_output}" "APT package-removal dependency preflight passed"

if preflight_failure=$(
  apt-get() {
    printf '%s\n' "containerd.io cannot be removed" >&2
    return 1
  }
  preflight_package_removal 2>&1
); then
  fail "APT dependency preflight unexpectedly accepted a resolver failure"
fi
assert_contains "${preflight_failure}" "before Docker or Swarm was changed"
assert_contains "${preflight_failure}" "containerd.io cannot be removed"

package_plan=$(remove_packages)
assert_contains "${package_plan}" \
  "apt-get purge -y --allow-change-held-packages docker-ce docker-ce-cli"

package_manager=dnf
package_plan=$(remove_packages)
assert_contains "${package_plan}" "dnf -y remove docker-ce docker-ce-cli"

package_manager=yum
package_plan=$(remove_packages)
assert_contains "${package_plan}" "yum -y remove docker-ce docker-ce-cli"

package_manager=zypper
package_plan=$(remove_packages)
assert_contains "${package_plan}" "zypper --non-interactive remove docker-ce docker-ce-cli"

package_manager=pacman
package_plan=$(remove_packages)
assert_contains "${package_plan}" "pacman -Rns --noconfirm docker-ce docker-ce-cli"

package_manager=apk
package_plan=$(remove_packages)
assert_contains "${package_plan}" "apk del docker-ce docker-ce-cli"

dry_run=false
purge_containerd_data=true
shared_containerd_detected=true
force_shared_containerd=false
if (validate_safety >/dev/null 2>&1); then
  fail "allowed shared containerd data removal without explicit force"
fi

dry_run=true
shared_runtime_warning=$(validate_safety 2>&1)
assert_contains "${shared_runtime_warning}" "a real containerd purge requires --force-shared-containerd"

swarm_state=inactive
docker_daemon_available=false
purge_static_binaries=false
purge_config=false
purge_data=false
leave_swarm || fail "non-Swarm host did not skip Swarm leave successfully"
remove_static_binaries || fail "disabled static binary purge did not return success"
remove_config || fail "disabled config purge did not return success"
remove_data || fail "disabled data purge did not return success"

if ! (
  # Isolate the post-uninstall assertion from tools preinstalled on the test runner.
  # shellcheck disable=SC2123
  PATH=""
  dry_run=false
  package_installed() {
    return 1
  }
  verify_uninstall
); then
  fail "clean post-uninstall state did not pass verification"
fi
if (
  # Isolate the post-uninstall assertion from tools preinstalled on the test runner.
  # shellcheck disable=SC2123
  PATH=""
  dry_run=false
  package_installed() {
    [[ "$1" == "docker-ce" ]]
  }
  verify_uninstall >/dev/null 2>&1
); then
  fail "post-uninstall verification accepted a remaining Docker package"
fi

printf '%s\n' 'node Docker uninstaller tests passed'
