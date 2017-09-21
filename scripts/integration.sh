#!/usr/bin/env bash
set -euo pipefail
set -x

unamestr=`uname`
# checks rhe operating system to issue readlink on Linux and 
# stat command on Mac OS.
if [[ "$unamestr" == 'Linux' ]]; then
	readlinkalias='readlink '
elif [[ "$unamestr" == 'Darwin' ]]; then
        readlinkalias='stat '
fi

export ROOT=$(dirname $($readlinkalias -f ${BASH_SOURCE%/*}))
if [ ! -f "$ROOT/.bin/ginkgo" ]; then
  (cd "$ROOT/src/staticfile/vendor/github.com/onsi/ginkgo/ginkgo/" && go install)
fi
if [ ! -f "$ROOT/.bin/buildpack-packager" ]; then
  (cd "$ROOT/src/staticfile/vendor/github.com/cloudfoundry/libbuildpack/packager/buildpack-packager" && go install)
fi

GINKGO_NODES=${GINKGO_NODES:-3}
GINKGO_ATTEMPTS=${GINKGO_ATTEMPTS:-2}

cd $ROOT/src/staticfile/integration

echo "Run Uncached Buildpack"
ginkgo -r --flakeAttempts=$GINKGO_ATTEMPTS -nodes $GINKGO_NODES -- --cached=false

echo "Run Cached Buildpack"
ginkgo -r --flakeAttempts=$GINKGO_ATTEMPTS -nodes $GINKGO_NODES -- --cached
