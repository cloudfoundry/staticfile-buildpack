#!/usr/bin/env bash
set -euo pipefail
set -x

export ROOT=$(dirname $(readlink -f ${BASH_SOURCE%/*}))
if [ ! -f "$ROOT/.bin/ginkgo" ]; then
  (cd "$ROOT/src/staticfile/vendor/github.com/onsi/ginkgo/ginkgo/" && go install)
fi
if [ ! -f "$ROOT/.bin/buildpack-packager" ]; then
  (cd "$ROOT/src/staticfile/vendor/github.com/cloudfoundry/libbuildpack/packager/buildpack-packager" && go install)
fi

GINKGO_NODES=${GINKGO_NODES:-3}
GINKGO_ATTEMPTS=${GINKGO_ATTEMPTS:-2}

cd $ROOT/src/staticfile/integration
if [ "${CACHED:-true}" = "false" ]; then
  ginkgo -r --flakeAttempts=$GINKGO_ATTEMPTS -nodes $GINKGO_NODES -- --cached=false
else
  ginkgo -r --flakeAttempts=$GINKGO_ATTEMPTS -nodes $GINKGO_NODES -- --cached
fi
