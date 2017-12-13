#!/usr/bin/env bash
set -exuo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."
source .envrc

GOOS=linux go build -ldflags="-s -w" -o bin/supply staticfile/supply/cli
GOOS=linux go build -ldflags="-s -w" -o bin/finalize staticfile/finalize/cli
