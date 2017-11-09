#!/usr/bin/env bash
set -euo pipefail

export ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd )"
cd $ROOT
source .envrc

if [ ! -f $ROOT/.bin/ginkgo ]; then
  (cd $ROOT/src/staticfile/vendor/github.com/onsi/ginkgo/ginkgo/ && go install)
fi

cd $ROOT/src/staticfile/
ginkgo -r -skipPackage=integration
