#!/usr/bin/env bash
set -ex

ROOTDIR="$( dirname "$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )" )"
BINDIR=$ROOTDIR/bin

GOPATH=$ROOTDIR GOOS=linux go build -o $BINDIR/compile compile

