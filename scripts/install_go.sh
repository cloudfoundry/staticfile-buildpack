#!/bin/bash
set -euo pipefail

GO_VERSION="1.11.4"

export GoInstallDir="/tmp/go$GO_VERSION"
mkdir -p $GoInstallDir

if [ ! -f $GoInstallDir/go/bin/go ]; then
  GO_SHA256="fb26c30e6a04ad937bbc657a1b5bba92f80096af1e8ee6da6430c045a8db3a5b"
  URL=https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz

  echo "-----> Download go ${GO_VERSION}"
  curl -s -L --retry 15 --retry-delay 2 $URL -o /tmp/go.tar.gz

  DOWNLOAD_SHA256=$(shasum -a 256 /tmp/go.tar.gz | cut -d ' ' -f 1)

  if [[ $DOWNLOAD_SHA256 != $GO_SHA256 ]]; then
    echo "       **ERROR** SHA256 mismatch: got $DOWNLOAD_SHA256 expected $GO_SHA256"
    exit 1
  fi

  tar xzf /tmp/go.tar.gz -C $GoInstallDir
  rm /tmp/go.tar.gz
fi
if [ ! -f $GoInstallDir/go/bin/go ]; then
  echo "       **ERROR** Could not download go"
  exit 1
fi
