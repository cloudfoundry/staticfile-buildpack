#!/bin/bash

function main() {
  set -e

  local dir
  dir="${1}"
  echo "dir -> ${dir}"

  mkdir -p "${dir}/dynatrace/oneagent/agent/lib64"

  curl -s --fail "http://{{.URI}}/manifest.json" > "${dir}/dynatrace/oneagent/manifest.json"
  curl -s --fail "http://{{.URI}}/dynatrace-env.sh" > "${dir}/dynatrace/oneagent/dynatrace-env.sh"
  curl -s --fail "http://{{.URI}}/liboneagentproc.so" > "${dir}/dynatrace/oneagent/agent/lib64/liboneagentproc.so"
}

main "${@}"
