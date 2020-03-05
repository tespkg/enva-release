#!/bin/bash

SCRIPT_PATH="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

cd ${SCRIPT_PATH}/alpine3.10 && docker build -t registry.tespkg.in/library/alpine:3.10 .
cd ${SCRIPT_PATH}/golang1.13-alpine3.10 && docker build -t registry.tespkg.in/library/golang:1.13-alpine3.10 .
cd ${SCRIPT_PATH}/nginx-alpine && docker build -t registry.tespkg.in/library/nginx:alpine .
cd ${SCRIPT_PATH}/node10.19.0-alpine3.10 && docker build -t registry.tespkg.in/library/node:10.19.0-alpine3.10 .
