#!/usr/bin/env bash

set -e
set -u
set -o pipefail

ROOTDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly ROOTDIR

function main() {
  local src
  src="$(find "${ROOTDIR}/src" -mindepth 1 -maxdepth 1 -type d )"

  IFS=" " read -r -a binaries <<< "$(find "${src}" -name cli -type d -print0 | xargs -0)"

  for path in "${binaries[@]}"; do
    local name
    name="$(basename "$(dirname "${path}")")"

    GOOS=linux \
      go build \
        -mod vendor \
        -ldflags="-s -w" \
        -o "${ROOTDIR}/bin/${name}" \
          "${path}"
  done

  if [[ -f "${ROOTDIR}/.windows" ]]; then
    for path in "${binaries[@]}"; do
      local name
      name="$(basename "$(dirname "${path}")")"

      GOOS=windows \
        go build \
          -mod vendor \
          -ldflags="-s -w" \
          -o "${ROOTDIR}/bin/${name}.exe" \
            "${path}"
    done
  fi
}

main "${@:-}"
