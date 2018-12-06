#!/bin/bash
set -euo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."
source .envrc

if [ ! -f .bin/ginkgo ]; then
  go get -u github.com/onsi/ginkgo/ginkgo
fi
if [ ! -f .bin/buildpack-packager ]; then
  go install github.com/cloudfoundry/libbuildpack/packager/buildpack-packager
fi
