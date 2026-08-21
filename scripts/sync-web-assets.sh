#!/bin/sh
# SPDX-License-Identifier: AGPL-3.0-only

set -eu

project_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
source_dir="${project_dir}/web/dist"
target_dir="${project_dir}/internal/webassets/dist"

if [ ! -f "${source_dir}/index.html" ]; then
  printf '%s\n' "web/dist/index.html is missing; run the Web production build first" >&2
  exit 1
fi
if [ ! -d "${target_dir}" ]; then
  printf '%s\n' "embedded asset target is missing: ${target_dir}" >&2
  exit 1
fi

find "${target_dir}" -type f ! -name .gitkeep -delete
cp -R "${source_dir}/." "${target_dir}/"
