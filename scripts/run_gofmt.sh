#!/bin/bash

set -e
set -o errexit
set -o pipefail
shopt -s nullglob


SCRIPTPATH="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROOTDIR=$(dirname "${SCRIPTPATH}")
cd "${ROOTDIR}"

function print_real_go_files {
    grep --files-without-match 'DO NOT EDIT' $(find . -iname '*.go' | grep -v vendor)
}

function goimports_all {
    echo "Running goimports"
    goimports -l -w $(print_real_go_files)
    return $?
}


goimports_all
echo "returning $?"
