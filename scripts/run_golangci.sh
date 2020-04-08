#!/bin/bash

set -e
set -o errexit
set -o pipefail
shopt -s nullglob

SCRIPTPATH="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROOTDIR=$(dirname "${SCRIPTPATH}")
cd "${ROOTDIR}"

if [[ "$1" == "--fix" ]]
then
    FIX="--fix"
fi

# if you want to update this version, also change the version number in .golangci.yml
GOLANGCI_VERSION="v1.22.2"
if [[ ! -f "$ROOTDIR"/.bin/golangci-lint ]] ; then
    curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b "$ROOTDIR"/.bin "$GOLANGCI_VERSION"
fi

"$ROOTDIR"/.bin/golangci-lint --version
env GOGC=25 "$ROOTDIR"/.bin/golangci-lint run ${FIX} -j 8 -v ./...
