#!/usr/bin/env bash

set -e
set -u
set -o pipefail

# shellcheck source=SCRIPTDIR/print.sh
source "$(dirname "${BASH_SOURCE[0]}")/print.sh"

function util::tools::path::export() {
  local dir
  dir="${1}"

  if ! echo "${PATH}" | grep -q "${dir}"; then
    PATH="${dir}:$PATH"
    export PATH
  fi
}

function util::tools::ginkgo::install() {
  local dir
  while [[ "${#}" != 0 ]]; do
    case "${1}" in
      --directory)
        dir="${2}"
        shift 2
        ;;

      *)
        util::print::error "unknown argument \"${1}\""
    esac
  done

  mkdir -p "${dir}"
  util::tools::path::export "${dir}"

  if [[ ! -f "${dir}/ginkgo" ]]; then
    util::print::title "Installing ginkgo"

    GOBIN="${dir}" \
      go get \
        -u \
        github.com/onsi/ginkgo/ginkgo
  fi
}

function util::tools::buildpack-packager::install() {
  local dir
  while [[ "${#}" != 0 ]]; do
    case "${1}" in
      --directory)
        dir="${2}"
        shift 2
        ;;

      *)
        util::print::error "unknown argument \"${1}\""
    esac
  done

  mkdir -p "${dir}"
  util::tools::path::export "${dir}"

  if [[ ! -f "${dir}/buildpack-packager" ]]; then
    util::print::title "Installing buildpack-packager"

    GOBIN="${dir}" \
      go install \
        github.com/cloudfoundry/libbuildpack/packager/buildpack-packager
  fi
}

function util::tools::jq::install() {
  local dir
  while [[ "${#}" != 0 ]]; do
    case "${1}" in
      --directory)
        dir="${2}"
        shift 2
        ;;

      *)
        util::print::error "unknown argument \"${1}\""
    esac
  done

  mkdir -p "${dir}"
  util::tools::path::export "${dir}"

  local os
  case "$(uname)" in
    "Darwin")
      os="osx-amd64"
      ;;

    "Linux")
      os="linux64"
      ;;

    *)
      echo "Unknown OS \"$(uname)\""
      exit 1
  esac

  if [[ ! -f "${dir}/jq" ]]; then
    util::print::title "Installing jq"

    curl "https://github.com/stedolan/jq/releases/download/jq-1.6/jq-${os}" \
      --silent \
      --location \
      --output "${dir}/jq"
    chmod +x "${dir}/jq"
  fi
}
