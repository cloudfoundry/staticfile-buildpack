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

  util::tools::jq::install --directory "${ROOTDIR}/.bin"

  IFS=" " read -r -a oses <<< "$(jq -r -S '.oses[]' "${ROOTDIR}/config.json" | xargs)"
  IFS=" " read -r -a binaries <<< "$(find "${src}" -name cli -type d -print0 | xargs -0)"

  for os in "${oses[@]}"; do
    for path in "${binaries[@]}"; do
      local name output
      name="$(basename "$(dirname "${path}")")"
      output="${ROOTDIR}/bin/${name}"

      if [[ "${os}" == "windows" ]]; then
        output="${output}.exe"
      fi

      GOOS="${os}" \
        go build \
          -mod vendor \
          -ldflags="-s -w" \
          -o "${output}" \
            "${path}"
    done
  done
}

main "${@:-}"
