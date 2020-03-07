#!/bin/bash

docker build -t registry.tespkg.in/library/enva:alpine3.10 .
docker build -t registry.tespkg.in/library/envs:alpine3.10 -f Dockerfile-envs .
