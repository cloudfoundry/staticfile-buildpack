#!/usr/bin/env bash

set -e
set -u
set -o pipefail

ROOTDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly ROOTDIR

source "${ROOTDIR}/.envrc"

function main() {
  local src
  src="$(find "${ROOTDIR}/src" -mindepth 1 -maxdepth 1 -type d )"

  "${ROOTDIR}/scripts/install_tools.sh"

  echo "Run Uncached Buildpack"
  CF_STACK="${CF_STACK:-cflinuxfs3}" \
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
  CF_STACK="${CF_STACK:-cflinuxfs3}" \
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
