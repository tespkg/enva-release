#!/bin/bash

docker run -d --network host -p 7001:7001 \
    -e ENVS_HTTP_ADDR=http://127.0.0.1:9112 \
    -e TARGET_SYS_ADMIN=CiQwOGE4Njg0Yi1kYjg4LTRiNzMtOTBhOS0zY2QxNjYxZjU0NjYSBWxvY2Fs \
    --rm --name ac registry.gitlab.com/target-digital-transformation/access-control/ac-be/master \
    /usr/local/bin/ac-serve serve \
    --address=:7001 \
    --oidc='${env:// .ssoIssuer }' \
    --dsn='${env:// .acDSN }' \
    --skip-client-id \
    --client-id=access-control \
    --client-secret=DzXZxyDObSpsnR7qLqQ4p1LEVoIiE49e \
    --redirect-uri='${env:// .acRESTAddr }'/oauth2
