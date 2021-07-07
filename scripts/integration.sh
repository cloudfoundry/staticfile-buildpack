#!/usr/bin/env bash

set -e
set -u
set -o pipefail

ROOTDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly ROOTDIR

# shellcheck source=SCRIPTDIR/.util/tools.sh
source "${ROOTDIR}/scripts/.util/tools.sh"

function main() {
  local src stack harness
  src="$(find "${ROOTDIR}/src" -mindepth 1 -maxdepth 1 -type d )"
  stack="$(jq -r -S .stack "${ROOTDIR}/config.json")"
  harness="$(jq -r -S .integration.harness "${ROOTDIR}/config.json")"

  IFS=$'\n' read -r -d '' -a matrix < <(
    jq -r -S -c .integration.matrix[] "${ROOTDIR}/config.json" \
      && printf "\0"
  )

  util::tools::ginkgo::install --directory "${ROOTDIR}/.bin"
  util::tools::buildpack-packager::install --directory "${ROOTDIR}/.bin"
  util::tools::cf::install --directory "${ROOTDIR}/.bin"

  for row in "${matrix[@]}"; do
    local cached parallel
    cached="$(jq -r -S .cached <<<"${row}")"
    parallel="$(jq -r -S .parallel <<<"${row}")"

    echo "Running integration suite (cached: ${cached}, parallel: ${parallel})"

    specs::run "${harness}" "${cached}" "${parallel}"
  done
}

function specs::run() {
  local harness cached parallel
  harness="${1}"
  cached="${2}"
  parallel="${3}"

  local nodes cached_flag serial_flag
  cached_flag="--cached=${cached}"
  serial_flag="-serial=true"
  nodes=1

  if [[ "${parallel}" == "true" ]]; then
    nodes=3
    serial_flag=""
  fi

  if [[ "${harness}" == "gotest" ]]; then
    specs::gotest::run "${nodes}" "${cached_flag}" "${serial_flag}"
  else
    specs::ginkgo::run "${nodes}" "${cached_flag}" "${serial_flag}"
  fi
}

function specs::gotest::run() {
  local nodes cached_flag serial_flag
  nodes="${1}"
  cached_flag="${2}"
  serial_flag="${3}"

  CF_STACK="${CF_STACK:-"${stack}"}" \
  BUILDPACK_FILE="${UNCACHED_BUILDPACK_FILE:-}" \
  GOMAXPROCS="${GOMAXPROCS:-"${nodes}"}" \
    go test \
      -count=1 \
      -timeout=0 \
      -mod vendor \
      -v \
        "${src}/integration" \
         "${cached_flag}" \
         "${serial_flag}"
}

function specs::ginkgo::run(){
  local nodes cached_flag serial_flag
  nodes="${1}"
  cached_flag="${2}"
  serial_flag="${3}"

  CF_STACK="${CF_STACK:-"${stack}"}" \
  BUILDPACK_FILE="${UNCACHED_BUILDPACK_FILE:-}" \
    ginkgo \
      -r \
      -mod vendor \
      --flakeAttempts "${GINKGO_ATTEMPTS:-2}" \
      -nodes "${nodes}" \
      --slowSpecThreshold 60 \
        "${src}/integration" \
      -- "${cached_flag}" "${serial_flag}"
}

main "${@:-}"
