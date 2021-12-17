#!/bin/bash

set -e
set -o errexit
set -o pipefail
shopt -s nullglob

SCRIPT_PATH="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROOT_DIR=$(dirname "${SCRIPT_PATH}")
ASSETS_DOCKERFILE_DIR=${ROOT_DIR}/assets/dockerfiles
BUILD_ARGS="--build-arg http_proxy=${http_proxy} --build-arg https_proxy=${http_proxy} --build-arg no_proxy=${no_proxy}"

echo  "Build base images..."
cd "${ROOT_DIR}"
docker build ${BUILD_ARGS} -t registry.tespkg.in/library/enva:alpine3.10 -f Dockerfile-enva .
docker build ${BUILD_ARGS} -t registry.tespkg.in/library/enva:debian-buster-slim -f Dockerfile-enva-debian .
docker build ${BUILD_ARGS} -t registry.tespkg.in/library/envs:alpine3.10 .

echo "Wrap vendor official images..."
cd ${ASSETS_DOCKERFILE_DIR}/alpine3.10 && docker build ${BUILD_ARGS} -t registry.tespkg.in/library/alpine:3.10 .
cd ${ASSETS_DOCKERFILE_DIR}/golang1.13-alpine3.10 && docker build ${BUILD_ARGS} -t registry.tespkg.in/library/golang:1.13-alpine3.10 .
cd ${ASSETS_DOCKERFILE_DIR}/golang1.13-buster && docker build ${BUILD_ARGS} -t registry.tespkg.in/library/golang:1.13-buster .
cd ${ASSETS_DOCKERFILE_DIR}/golang1.13-buster-orcl && docker build ${BUILD_ARGS} -t registry.tespkg.in/library/golang:1.13-buster-orcl .
cd ${ASSETS_DOCKERFILE_DIR}/golang1.14 && docker build ${BUILD_ARGS} -t registry.tespkg.in/library/golang:1.14 .
cd ${ASSETS_DOCKERFILE_DIR}/nginx-alpine && docker build ${BUILD_ARGS} -t registry.tespkg.in/library/nginx:alpine .
cd ${ASSETS_DOCKERFILE_DIR}/node16-alpine3.13 && docker build ${BUILD_ARGS} -t registry.tespkg.in/library/node:16-alpine3.13 .
cd ${ASSETS_DOCKERFILE_DIR}/debian-buster-slim && docker build ${BUILD_ARGS} -t registry.tespkg.in/library/debian:buster-slim .
cd ${ASSETS_DOCKERFILE_DIR}/debian-buster-slim-orcl && docker build ${BUILD_ARGS} -t registry.tespkg.in/library/debian:buster-slim-orcl .
cd ${ASSETS_DOCKERFILE_DIR}/debian-bullseye-slim && docker build ${BUILD_ARGS} -t registry.tespkg.in/library/debian:bullseye-slim .
cd ${ASSETS_DOCKERFILE_DIR}/ubuntu18.04 && docker build ${BUILD_ARGS} -t registry.tespkg.in/library/ubuntu:18.04 .
cd ${ASSETS_DOCKERFILE_DIR}/openjdk-8-jre-slim && docker build ${BUILD_ARGS} -t registry.tespkg.in/library/openjdk:8-jre-slim .
cd ${ASSETS_DOCKERFILE_DIR}/openjdk-11-jre-slim && docker build ${BUILD_ARGS} -t registry.tespkg.in/library/openjdk:11-jre-slim .
cd ${ASSETS_DOCKERFILE_DIR}/python-3.6.10 && docker build ${BUILD_ARGS} -t registry.tespkg.in/library/python:3.6.10 .
cd ${ASSETS_DOCKERFILE_DIR}/python-3.6.10-alpine3.10 && docker build ${BUILD_ARGS} -t registry.tespkg.in/library/python:3.6.10-alpine3.10 .
cd ${ASSETS_DOCKERFILE_DIR}/mongodb-bi-connector && docker build ${BUILD_ARGS} -t registry.tespkg.in/library/mongodb-bi-connector .

if [[ $# == 1 ]] && [[ $1 == "true" ]]; then
    echo "Push base images..."
    docker push registry.tespkg.in/library/enva:alpine3.10
    docker push registry.tespkg.in/library/enva:debian-buster-slim
    docker push registry.tespkg.in/library/envs:alpine3.10

    echo "Push wrapped vendor images..."
    docker push registry.tespkg.in/library/alpine:3.10
    docker push registry.tespkg.in/library/golang:1.13-alpine3.10
    docker push registry.tespkg.in/library/golang:1.13-buster
    docker push registry.tespkg.in/library/golang:1.13-buster-orcl
    docker push registry.tespkg.in/library/golang:1.14
    docker push registry.tespkg.in/library/nginx:alpine
    docker push registry.tespkg.in/library/node:16-alpine3.13
    docker push registry.tespkg.in/library/debian:buster-slim
    docker push registry.tespkg.in/library/debian:bullseye-slim
    docker push registry.tespkg.in/library/debian:buster-slim-orcl
    docker push registry.tespkg.in/library/ubuntu:18.04
    docker push registry.tespkg.in/library/openjdk:8-jre-slim
    docker push registry.tespkg.in/library/openjdk:11-jre-slim
    docker push registry.tespkg.in/library/python:3.6.10
    docker push registry.tespkg.in/library/python:3.6.10-alpine3.10
    docker push registry.tespkg.in/library/mongodb-bi-connector
fi

