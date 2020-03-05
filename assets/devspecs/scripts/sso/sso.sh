#!/bin/bash

docker run --network meera -p 5556:5556 -p 5557:5557 \
    -e ENVS_HTTP_ADDR=http://host.docker.internal:9112 \
    --name sso registry.tespkg.in/sso/master \
    /usr/local/bin/dex serve '$${envf:// .ssoConfig | default /usr/local/config/sso.yaml }'
