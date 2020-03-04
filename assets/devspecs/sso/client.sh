#!/bin/bash

docker run -d --network meera -p 5555:5555 \
    -e ENVS_HTTP_ADDR=http://host.docker.internal:9112 \
    --rm --name sso-client registry.gitlab.com/target-digital-transformation/sso/client \
    /usr/local/bin/dex-client \
    --listen=http://0.0.0.0:5555 \
    --issuer='${env:// .ssoIssuer }' \
    --redirect-uri='${env:// .ssoClientAddr }/callback'
