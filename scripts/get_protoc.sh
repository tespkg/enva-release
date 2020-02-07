#!/bin/bash

set -o errexit
set -o pipefail
shopt -s nullglob

root="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd )"
bindir=${root}/.bin

GOOGLEAPIS_REPO=https://github.com/googleapis/googleapis
GOGOGOOGLEAPIS_REPO=https://github.com/gogo/googleapis

export GOBIN=${bindir}
export GO111MODULE=on

# install protoc & related command line tools
function install_binaries(){
    dist=${bindir}/protoc

    # if the command exist. don`t fetch
    if [[ -f ${dist} ]] ; then
        return
    fi

    # get protoc
    VERSION="3.7.1"

    # use the go tool to determine OS.
    OS=$( go env GOOS )
    if [[ "$OS" = "darwin" ]]; then
      OS="osx"
    fi

    rm -rf ${bindir}
    mkdir -p ${bindir}
    ZIP="protoc-${VERSION}-${OS}-x86_64.zip"
    URL="https://github.com/google/protobuf/releases/download/v${VERSION}/${ZIP}"

    wget ${URL} -O ${ZIP}

    # Unpack the protoc.
    unzip ${ZIP} -d /tmp/protoc
    mkdir -p ${bindir}/include
    mv /tmp/protoc/include/* ${bindir}/include/
    mv /tmp/protoc/bin/protoc ${bindir}/
    chmod +x ${bindir}/protoc
    rm -rf /tmp/protoc
    rm ${ZIP}

    # install golang proto plugin
    go install -v github.com/golang/protobuf/protoc-gen-go
    go install -v github.com/gogo/protobuf/protoc-gen-gogofast
    go install -v github.com/gogo/protobuf/protoc-gen-gogoslick
    go install -v github.com/golang/mock/mockgen
}

function die() {
  echo 1>&2 $*
  exit 1
}

# Sanity check that the right tools are accessible.
for tool in go git unzip; do
  q=$(which ${tool}) || die "${tool} not found"
  echo 1>&2 "$tool: $q"
done

install_binaries

remove_dirs=
trap 'rm -rf ${remove_dirs}' EXIT

if [[ ! -d "${bindir}/googleapis" ]]; then
  apidir=$(mktemp -d -t regen-cds-api.XXXXXX)
  git clone ${GOOGLEAPIS_REPO} ${apidir}
  remove_dirs=${apidir}
  mkdir -p "${bindir}"
  cp -rf ${apidir} ${bindir}/googleapis
fi

if [[ ! -d "${bindir}/github.com/gogo/googleapis" ]]; then
  apidir=$(mktemp -d -t regen-cds-api.XXXXXX)
  git clone ${GOGOGOOGLEAPIS_REPO} ${apidir}
  remove_dirs="$remove_dirs"
  mkdir -p "${bindir}/github.com/gogo"
  cp -rf ${apidir} "${bindir}/github.com/gogo/googleapis"
fi

wait

echo 1>&2 "get-protoc done"

