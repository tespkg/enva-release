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
docker build --build-arg http_proxy=${http_proxy} --build-arg https_proxy=${http_proxy} --build-arg no_proxy=${no_proxy} -t registry.tespkg.in/library/enva:alpine3.10 -f Dockerfile-enva .
docker build --build-arg http_proxy=${http_proxy} --build-arg https_proxy=${http_proxy} --build-arg no_proxy=${no_proxy} -t registry.tespkg.in/library/enva:debian-buster-slim -f Dockerfile-enva-debian .

echo "Wrap vendor official images..."
cd ${ASSETS_DOCKERFILE_DIR}/alpine3.10 && docker build --build-arg http_proxy=${http_proxy} --build-arg https_proxy=${http_proxy} --build-arg no_proxy=${no_proxy} -t registry.tespkg.in/library/alpine:3.10 .
cd ${ASSETS_DOCKERFILE_DIR}/golang1.13-alpine3.10 && docker build --build-arg http_proxy=${http_proxy} --build-arg https_proxy=${http_proxy} --build-arg no_proxy=${no_proxy} -t registry.tespkg.in/library/golang:1.13-alpine3.10 .
cd ${ASSETS_DOCKERFILE_DIR}/golang1.13-buster && docker build --build-arg http_proxy=${http_proxy} --build-arg https_proxy=${http_proxy} --build-arg no_proxy=${no_proxy} -t registry.tespkg.in/library/golang:1.13-buster .
cd ${ASSETS_DOCKERFILE_DIR}/golang1.13-buster-orcl && docker build --build-arg http_proxy=${http_proxy} --build-arg https_proxy=${http_proxy} --build-arg no_proxy=${no_proxy} -t registry.tespkg.in/library/golang:1.13-buster-orcl .
cd ${ASSETS_DOCKERFILE_DIR}/nginx-alpine && docker build --build-arg http_proxy=${http_proxy} --build-arg https_proxy=${http_proxy} --build-arg no_proxy=${no_proxy} -t registry.tespkg.in/library/nginx:alpine .
cd ${ASSETS_DOCKERFILE_DIR}/debian-buster-slim && docker build --build-arg http_proxy=${http_proxy} --build-arg https_proxy=${http_proxy} --build-arg no_proxy=${no_proxy} -t registry.tespkg.in/library/debian:buster-slim .
cd ${ASSETS_DOCKERFILE_DIR}/ubuntu18.04 && docker build --build-arg http_proxy=${http_proxy} --build-arg https_proxy=${http_proxy} --build-arg no_proxy=${no_proxy} -t registry.tespkg.in/library/ubuntu:18.04 .

if [[ $# == 1 ]] && [[ $1 == "true" ]]; then
    echo "Push base images..."
    docker push registry.tespkg.in/library/enva:alpine3.10
    docker push registry.tespkg.in/library/enva:debian-buster-slim

    echo "Push wrapped vendor images..."
    docker push registry.tespkg.in/library/alpine:3.10
    docker push registry.tespkg.in/library/golang:1.13-alpine3.10
    docker push registry.tespkg.in/library/golang:1.13-buster
    docker push registry.tespkg.in/library/golang:1.13-buster-orcl
    docker push registry.tespkg.in/library/nginx:alpine
    docker push registry.tespkg.in/library/debian:buster-slim
    docker push registry.tespkg.in/library/ubuntu:18.04
fi

