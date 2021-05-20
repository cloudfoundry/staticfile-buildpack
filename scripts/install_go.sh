#!/bin/bash

set -e
set -u
set -o pipefail

function main() {
  if [[ "${CF_STACK:-}" != "cflinuxfs3" ]]; then
    echo "       **ERROR** Unsupported stack"
    echo "                 See https://docs.cloudfoundry.org/devguide/deploy-apps/stacks.html for more info"
    exit 1
  fi

  local version expected_sha dir
  version="1.15.5"
  expected_sha="fd04494f7a2dd478b0d31cb949aae7f154749cae1242581b1574f7e590b3b7e6"
  dir="/tmp/go${version}"

  mkdir -p "${dir}"

  if [[ ! -f "${dir}/go/bin/go" ]]; then
    local url
    url="https://buildpacks.cloudfoundry.org/dependencies/go/go_${version}_linux_x64_${CF_STACK}_${expected_sha:0:8}.tgz"

    echo "-----> Download go ${version}"
    curl "${url}" \
      --silent \
      --location \
      --retry 15 \
      --retry-delay 2 \
      --output "/tmp/go.tgz"

    local sha
    sha="$(shasum -a 256 /tmp/go.tgz | cut -d ' ' -f 1)"

    if [[ "${sha}" != "${expected_sha}" ]]; then
      echo "       **ERROR** SHA256 mismatch: got ${sha}, expected ${expected_sha}"
      exit 1
    fi

    tar xzf "/tmp/go.tgz" -C "${dir}"
    rm "/tmp/go.tgz"
  fi

  if [[ ! -f "${dir}/bin/go" ]]; then
    echo "       **ERROR** Could not download go"
    exit 1
  fi

  GoInstallDir="${dir}"
  export GoInstallDir
}

main "${@:-}"
