#!/bin/bash

set -e
set -u
set -o pipefail

ROOTDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly ROOTDIR

source "${ROOTDIR}/.envrc"

function main() {
  pushd "${ROOTDIR}" > /dev/null || return
    go get -u github.com/onsi/ginkgo/ginkgo

    if [[ ! -f "${ROOTDIR}/.bin/buildpack-packager" ]]; then
      go install github.com/cloudfoundry/libbuildpack/packager/buildpack-packager
    fi
  popd > /dev/null || return
}

main "${@:-}"
