#!/bin/bash

docker run -d --network host -p 5556:5556 -p 5557:5557 -e ENVS_HTTP_ADDR=http://127.0.0.1:9112 --rm --name sso registry.gitlab.com/target-digital-transformation/sso/master \
    /usr/local/bin/dex serve '${envf:// .ssoConfig }'
