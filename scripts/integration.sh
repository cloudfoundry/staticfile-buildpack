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
  util::tools::cf::install --directory "${ROOTDIR}/.bin"

  local stack
  stack="$(jq -r -S .stack "${ROOTDIR}/config.json")"

  cached=true
  serial=true
  if [[ "${src}" == *python ]]; then
    run_specs "uncached" "parallel"
    run_specs "uncached" "serial"

    run_specs "cached" "parallel"
    run_specs "cached" "serial"
  else
    run_specs "uncached" "parallel"
    run_specs "cached" "parallel"
  fi
}

function run_specs(){
  local cached serial nodes

  cached="false"
  serial=""
  nodes="${GINKGO_NODES:-3}"

  echo "Run ${1} Buildpack"

  if [[ "${1}" == "cached" ]] ; then
    cached="true"
  fi

  if [[ "${2}" == "serial" ]]; then
    nodes=1
    serial="-serial=true"
  fi

  CF_STACK="${CF_STACK:-"${stack}"}" \
  BUILDPACK_FILE="${UNCACHED_BUILDPACK_FILE:-}" \
    ginkgo \
      -r \
      -mod vendor \
      --flakeAttempts "${GINKGO_ATTEMPTS:-2}" \
      -nodes ${nodes} \
      --slowSpecThreshold 60 \
        "${src}/integration" \
      -- --cached="${cached}" ${serial}
}

main "${@:-}"
