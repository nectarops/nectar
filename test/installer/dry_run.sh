#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-only

set -Eeuo pipefail
IFS=$'\n\t'

readonly CENTOS_STREAM_9_IMAGE="quay.io/centos/centos@sha256:64e5a212e4f2e7b706dbd822968914bb8def7de0a7fdfd3bf248241f8758101c"
readonly CENTOS_STREAM_10_IMAGE="quay.io/centos/centos@sha256:ad14f7d919c9b9995a236abace06e888d37d6145e048a6ae26770e1f5dc718a8"

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
installer="${repo_root}/install.sh"
deploy_stack="${repo_root}/deploy/stack.yml"
centos_8_fixture="${repo_root}/test/installer/testdata/centos-8-os-release"
ip_overlap_fixture="${repo_root}/test/installer/testdata/ip-overlap"
existing_service_docker_fixture="${repo_root}/test/installer/testdata/docker-existing-service"

assert_contains() {
  local output=$1
  local expected=$2

  [[ "${output}" == *"${expected}"* ]] || {
    printf 'expected output to contain: %s\n\n%s\n' "${expected}" "${output}" >&2
    exit 1
  }
}

assert_not_contains() {
  local output=$1
  local unexpected=$2

  [[ "${output}" != *"${unexpected}"* ]] || {
    printf 'expected output not to contain: %s\n\n%s\n' "${unexpected}" "${output}" >&2
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
  assert_contains "${output}" "Would create or reuse dedicated Nectar Overlay network nectar_control (172.31.255.0/24)."
  assert_contains "${output}" "Nectar Web port 8080 will be published in host mode on the labeled Manager."
  assert_contains "${output}" "Dry run completed. No host changes were made."
}

command -v docker >/dev/null 2>&1 || {
  printf 'docker is required for installer distribution tests\n' >&2
  exit 1
}

installer_content=$(<"${installer}")
# Exiting awk after the first match sends SIGPIPE to apt-cache under pipefail and aborts installation with 141.
# shellcheck disable=SC2016
assert_not_contains "${installer_content}" '$3 ~ ("^5:" requested "-") {print $3; exit}'
# shellcheck disable=SC2016
assert_contains "${installer_content}" \
  '$3 ~ ("^5:" requested "-") && !selected {print $3; selected = 1}'
# These assertions intentionally match unexpanded template expressions in install.sh.
# shellcheck disable=SC2016
assert_contains "${installer_content}" \
  'traefik.http.routers.nectar.rule: "Host(\`${preserved_management_domain}\`)"'
assert_contains "${installer_content}" \
  'traefik.http.services.nectar.loadbalancer.server.port: "8080"'
# shellcheck disable=SC2016
assert_contains "${installer_content}" 'traefik.swarm.network: "${MANAGEMENT_NETWORK_NAME}"'
assert_contains "${installer_content}" $'  ${MANAGEMENT_NETWORK_NAME}:\n    external: true'

deploy_stack_content=$(<"${deploy_stack}")
assert_contains "${deploy_stack_content}" "mode: host"
assert_contains "${deploy_stack_content}" $'    networks:\n      - nectar_control'
assert_contains "${deploy_stack_content}" $'networks:\n  nectar_control:\n    external: true'
if [[ "${deploy_stack_content}" == *"mode: ingress"* ]]; then
  printf 'deploy/stack.yml must not publish Nectar through the Swarm ingress routing mesh\n' >&2
  exit 1
fi

run_supported_dry_run "${CENTOS_STREAM_9_IMAGE}" "9"
run_supported_dry_run "${CENTOS_STREAM_10_IMAGE}" "10"

explicit_subnet_output=$(docker run --rm \
  --volume "${installer}:/install.sh:ro" \
  "${CENTOS_STREAM_9_IMAGE}" \
  bash /install.sh --dry-run --advertise-addr 192.0.2.10 --network-subnet 172.30.255.0/24 2>&1)
assert_contains "${explicit_subnet_output}" \
  "Would create or reuse dedicated Nectar Overlay network nectar_control (172.30.255.0/24)."

if invalid_subnet_output=$(docker run --rm \
  --volume "${installer}:/install.sh:ro" \
  "${CENTOS_STREAM_9_IMAGE}" \
  bash /install.sh --dry-run --advertise-addr 192.0.2.10 --network-subnet 172.30.255.1/24 2>&1); then
  printf 'expected a non-network /24 address to fail\n%s\n' "${invalid_subnet_output}" >&2
  exit 1
fi
assert_contains "${invalid_subnet_output}" "Nectar network subnet must use the network address"

if public_subnet_output=$(docker run --rm \
  --volume "${installer}:/install.sh:ro" \
  "${CENTOS_STREAM_9_IMAGE}" \
  bash /install.sh --dry-run --advertise-addr 192.0.2.10 --network-subnet 203.0.113.0/24 2>&1); then
  printf 'expected a public network subnet to fail\n%s\n' "${public_subnet_output}" >&2
  exit 1
fi
assert_contains "${public_subnet_output}" "Nectar network subnet must use RFC 1918 private address space"

if overlapping_subnet_output=$(docker run --rm \
  --volume "${installer}:/install.sh:ro" \
  --volume "${ip_overlap_fixture}:/usr/local/bin/ip:ro" \
  "${CENTOS_STREAM_9_IMAGE}" \
  bash /install.sh --dry-run --advertise-addr 172.30.10.8 \
  --network-subnet 172.30.10.0/24 2>&1); then
  printf 'expected a subnet overlapping the host interface to fail\n%s\n' "${overlapping_subnet_output}" >&2
  exit 1
fi
assert_contains "${overlapping_subnet_output}" "overlaps existing address space"

if unsupported_output=$(docker run --rm \
  --volume "${installer}:/install.sh:ro" \
  --volume "${centos_8_fixture}:/etc/os-release:ro" \
  "${CENTOS_STREAM_9_IMAGE}" \
  bash /install.sh --dry-run --advertise-addr 192.0.2.10 2>&1); then
  printf 'expected CentOS Linux 8 dry run to fail\n%s\n' "${unsupported_output}" >&2
  exit 1
fi
assert_contains "${unsupported_output}" "supported CentOS releases are CentOS Stream 9 and 10"

configured_upgrade_output=$(docker run --rm \
  --env FAKE_SERVICE_EXISTS=true \
  --env FAKE_MANAGEMENT_DOMAIN=nectar.example.com \
  --volume "${installer}:/install.sh:ro" \
  --volume "${existing_service_docker_fixture}:/usr/local/bin/docker:ro" \
  "${CENTOS_STREAM_9_IMAGE}" \
  bash /install.sh --dry-run --advertise-addr 192.0.2.10 2>&1)
assert_contains "${configured_upgrade_output}" \
  "Preserving HTTPS management route for nectar.example.com during the service update."

unconfigured_upgrade_output=$(docker run --rm \
  --env FAKE_SERVICE_EXISTS=true \
  --volume "${installer}:/install.sh:ro" \
  --volume "${existing_service_docker_fixture}:/usr/local/bin/docker:ro" \
  "${CENTOS_STREAM_9_IMAGE}" \
  bash /install.sh --dry-run --advertise-addr 192.0.2.10 2>&1)
assert_not_contains "${unconfigured_upgrade_output}" "Preserving HTTPS management route"

if invalid_management_network_output=$(docker run --rm \
  --env FAKE_SERVICE_EXISTS=true \
  --env FAKE_MANAGEMENT_DOMAIN=nectar.example.com \
  --env FAKE_MANAGEMENT_NETWORK_DRIVER=bridge \
  --env FAKE_MANAGEMENT_NETWORK_SCOPE=local \
  --volume "${installer}:/install.sh:ro" \
  --volume "${existing_service_docker_fixture}:/usr/local/bin/docker:ro" \
  "${CENTOS_STREAM_9_IMAGE}" \
  bash /install.sh --dry-run --advertise-addr 192.0.2.10 2>&1); then
  printf 'expected invalid management network state to fail\n%s\n' \
    "${invalid_management_network_output}" >&2
  exit 1
fi
assert_contains "${invalid_management_network_output}" \
  "traefik-public is not a Swarm overlay network"

printf 'installer distribution dry-run tests passed\n'
