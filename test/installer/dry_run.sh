#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-only

set -Eeuo pipefail
IFS=$'\n\t'

readonly CENTOS_STREAM_9_IMAGE="quay.io/centos/centos@sha256:64e5a212e4f2e7b706dbd822968914bb8def7de0a7fdfd3bf248241f8758101c"
readonly CENTOS_STREAM_10_IMAGE="quay.io/centos/centos@sha256:ad14f7d919c9b9995a236abace06e888d37d6145e048a6ae26770e1f5dc718a8"

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
installer="${repo_root}/install.sh"
centos_8_fixture="${repo_root}/test/installer/testdata/centos-8-os-release"

assert_contains() {
  local output=$1
  local expected=$2

  [[ "${output}" == *"${expected}"* ]] || {
    printf 'expected output to contain: %s\n\n%s\n' "${expected}" "${output}" >&2
    exit 1
  }
}

run_supported_dry_run() {
  local image=$1
  local version=$2
  local output

  output=$(docker run --rm \
    --volume "${installer}:/install.sh:ro" \
    "${image}" \
    bash /install.sh --dry-run --advertise-addr 192.0.2.10 2>&1)

  assert_contains "${output}" "Host: centos ${version}"
  assert_contains "${output}" "Docker's repository signing key fingerprint 060A61C51B558A7F742B77AAC52FEB6B621E9F35"
  assert_contains "${output}" "Would configure Docker's signed CentOS Stream repository for release ${version}."
  assert_contains "${output}" "[dry-run] dnf -y install ca-certificates gnupg2"
  assert_contains "${output}" "Dry run completed. No host changes were made."
}

command -v docker >/dev/null 2>&1 || {
  printf 'docker is required for installer distribution tests\n' >&2
  exit 1
}

run_supported_dry_run "${CENTOS_STREAM_9_IMAGE}" "9"
run_supported_dry_run "${CENTOS_STREAM_10_IMAGE}" "10"

if unsupported_output=$(docker run --rm \
  --volume "${installer}:/install.sh:ro" \
  --volume "${centos_8_fixture}:/etc/os-release:ro" \
  "${CENTOS_STREAM_9_IMAGE}" \
  bash /install.sh --dry-run --advertise-addr 192.0.2.10 2>&1); then
  printf 'expected CentOS Linux 8 dry run to fail\n%s\n' "${unsupported_output}" >&2
  exit 1
fi
assert_contains "${unsupported_output}" "supported CentOS releases are CentOS Stream 9 and 10"

printf 'installer distribution dry-run tests passed\n'
