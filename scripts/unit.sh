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

  ginkgo \
    -r \
    -mod vendor \
    -skipPackage brats,integration \
      "${src}/..."
}

main "${@:-}"
