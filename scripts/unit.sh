#!/usr/bin/env bash
set -euo pipefail
unamestr=`uname`
# checks rhe operating system to issue readlink on Linux and 
# stat command on Mac OS.
if [[ "$unamestr" == 'Linux' ]]; then
        readlinkalias='readlink '
elif [[ "$unamestr" == 'Darwin' ]]; then
        readlinkalias='stat '
fi

export ROOT=`dirname $($readlinkalias -f ${BASH_SOURCE%/*})`
if [ ! -f $ROOT/.bin/ginkgo ]; then
  (cd $ROOT/src/staticfile/vendor/github.com/onsi/ginkgo/ginkgo/ && go install)
fi

cd $ROOT/src/staticfile/
ginkgo -r -skipPackage=integration
