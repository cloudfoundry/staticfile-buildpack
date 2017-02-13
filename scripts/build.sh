#!/usr/bin/env bash

ROOTDIR="$( dirname "$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )" )"
BINDIR=$ROOTDIR/bin

if [[ ! -e $ROOTDIR/dependencies ]]; then
  echo "building uncached buildpack"
  exit 0
fi

set -ex

GOPATH=$ROOTDIR GOOS=linux go build -o $BINDIR/compile compile

