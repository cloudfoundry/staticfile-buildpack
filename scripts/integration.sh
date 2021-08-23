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
  stack="${CF_STACK:-$(jq -r -S .stack "${ROOTDIR}/config.json")}"
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

    specs::run "${harness}" "${cached}" "${parallel}" "${stack}"
  done
}

function specs::run() {
  local harness cached parallel stack
  harness="${1}"
  cached="${2}"
  parallel="${3}"
  stack="${4}"

  local nodes cached_flag serial_flag
  cached_flag="--cached=${cached}"
  serial_flag="-serial=true"
  nodes=1

  if [[ "${parallel}" == "true" ]]; then
    nodes=3
    serial_flag=""
  fi

  local buildpack_file
  buildpack_file="$(buildpack::package "1.2.3" "${cached}" "${stack}")"

  if [[ "${harness}" == "gotest" ]]; then
    specs::gotest::run "${nodes}" "${cached_flag}" "${serial_flag}" "${buildpack_file}" "${stack}"
  else
    specs::ginkgo::run "${nodes}" "${cached_flag}" "${serial_flag}" "${buildpack_file}" "${stack}"
  fi
}

function specs::gotest::run() {
  local nodes cached_flag serial_flag buildpack_file stack
  nodes="${1}"
  cached_flag="${2}"
  serial_flag="${3}"
  buildpack_file="${4}"
  stack="${5}"

  CF_STACK="${stack}" \
  BUILDPACK_FILE="${BUILDPACK_FILE:-"${buildpack_file}"}" \
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
  local nodes cached_flag serial_flag buildpack_file stack
  nodes="${1}"
  cached_flag="${2}"
  serial_flag="${3}"
  buildpack_file="${4}"
  stack="${5}"

  CF_STACK="${stack}" \
  BUILDPACK_FILE="${BUILDPACK_FILE:-"${buildpack_file}"}" \
    ginkgo \
      -r \
      -mod vendor \
      --flakeAttempts "${GINKGO_ATTEMPTS:-2}" \
      -nodes "${nodes}" \
      --slowSpecThreshold 60 \
        "${src}/integration" \
      -- "${cached_flag}" "${serial_flag}"
}

function buildpack::package() {
  local version cached stack
  version="${1}"
  cached="${2}"
  stack="${3}"

  local name cached_flag
  name="buildpack-v${version}-uncached.zip"
  cached_flag=""
  if [[ "${cached}" == "true" ]]; then
    cached_flag="--cached"
    name="buildpack-v${version}-cached.zip"
  fi

  local output
  output="$(mktemp -d)/${name}"

  bash "${ROOTDIR}/scripts/package.sh" \
    --version "${version}" \
    --output "${output}" \
    --stack "${stack}" \
    "${cached_flag}" > /dev/null

  printf "%s" "${output}"
}

main "${@:-}"
