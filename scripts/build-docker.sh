#!/bin/bash

set -e
set -o errexit
set -o pipefail
shopt -s nullglob

SCRIPT_PATH="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROOT_DIR=$(dirname "${SCRIPT_PATH}")
ASSETS_DOCKERFILE_DIR=${ROOT_DIR}/assets/dockerfiles

echo  "Build base images..."
cd "${ROOT_DIR}"
docker build -t registry.tespkg.in/library/enva:alpine3.10 .
docker build -t registry.tespkg.in/library/envs:alpine3.10 -f Dockerfile-envs .

echo "Wrap vendor official images..."
cd ${ASSETS_DOCKERFILE_DIR}/alpine3.10 && docker build -t registry.tespkg.in/library/alpine:3.10 .
cd ${ASSETS_DOCKERFILE_DIR}/golang1.13-alpine3.10 && docker build -t registry.tespkg.in/library/golang:1.13-alpine3.10 .
cd ${ASSETS_DOCKERFILE_DIR}/nginx-alpine && docker build -t registry.tespkg.in/library/nginx:alpine .

if [[ $# == 1 ]] && [[ $1 == "true" ]]; then
    echo "Push base images..."
    docker push registry.tespkg.in/library/enva:alpine3.10
    docker push registry.tespkg.in/library/envs:alpine3.10

    echo "Push wrapped vendor images..."
    docker push registry.tespkg.in/library/alpine:3.10
    docker push registry.tespkg.in/library/golang:1.13-alpine3.10
    docker push registry.tespkg.in/library/nginx:alpine
fi

