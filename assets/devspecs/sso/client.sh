#!/bin/bash

docker run -d --network host -p 5555:5555 -e ENVS_HTTP_ADDR=http://127.0.0.1:9112 --rm --name sso-client registry.gitlab.com/target-digital-transformation/sso/client \
    /usr/local/bin/dex-client --issuer '${env:// .ssoIssuer }'
