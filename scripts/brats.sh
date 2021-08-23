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

  if [[ ! -d "${src}/brats" ]]; then
    echo "There are no brats tests to run"
    exit 0
  fi

  util::tools::ginkgo::install --directory "${ROOTDIR}/.bin"
  util::tools::buildpack-packager::install --directory "${ROOTDIR}/.bin"
  util::tools::jq::install --directory "${ROOTDIR}/.bin"
  util::tools::cf::install --directory "${ROOTDIR}/.bin"

  local stack
  stack="$(jq -r -S .stack "${ROOTDIR}/config.json")"

  echo "Run Buildpack Runtime Acceptance Tests"
  CF_STACK="${CF_STACK:-"${stack}"}" \
    ginkgo \
      -r \
      -mod vendor \
      --flakeAttempts "${GINKGO_ATTEMPTS:-2}" \
      -nodes "${GINKGO_NODES:-3}" \
        "${src}/brats"
}

main "${@:-}"
