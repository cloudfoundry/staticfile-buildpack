#!/usr/bin/env bash

set -e
set -u
set -o pipefail

ROOTDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly ROOTDIR

# shellcheck source=SCRIPTDIR/.util/tools.sh
source "${ROOTDIR}/scripts/.util/tools.sh"

function main() {
  local src
  src="$(find "${ROOTDIR}/src" -mindepth 1 -maxdepth 1 -type d )"

  util::tools::ginkgo::install --directory "${ROOTDIR}/.bin"
  util::tools::buildpack-packager::install --directory "${ROOTDIR}/.bin"

  local stack
  stack="$(jq -r -S .stack "${ROOTDIR}/config.json")"

  echo "Run Uncached Buildpack"
  CF_STACK="${CF_STACK:-"${stack}"}" \
  BUILDPACK_FILE="${UNCACHED_BUILDPACK_FILE:-}" \
    ginkgo \
      -r \
      -mod vendor \
      --flakeAttempts "${GINKGO_ATTEMPTS:-2}" \
      -nodes "${GINKGO_NODES:-3}" \
      --slowSpecThreshold 60 \
        "${src}/integration" \
      -- --cached=false

  echo "Run Cached Buildpack"
  CF_STACK="${CF_STACK:-"${stack}"}" \
  BUILDPACK_FILE="${CACHED_BUILDPACK_FILE:-}" \
    ginkgo \
      -mod vendor \
      -r \
      --flakeAttempts "${GINKGO_ATTEMPTS:-2}" \
      -nodes "${GINKGO_NODES:-3}" \
      --slowSpecThreshold 60 \
        "${src}/integration" \
      -- --cached=true
}

main "${@:-}"
