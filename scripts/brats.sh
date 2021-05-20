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


  echo "Run Buildpack Runtime Acceptance Tests"

  CF_STACK="${CF_STACK:-cflinuxfs3}" \
    ginkgo \
      -r \
      -mod vendor \
      --flakeAttempts "${GINKGO_ATTEMPTS:-2}" \
      -nodes "${GINKGO_NODES:-3}" \
        "${src}/brats"
}

main "${@:-}"
