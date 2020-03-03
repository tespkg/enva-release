#!/bin/bash

docker run --network host -p 9301:9301 \
    -e ENVS_HTTP_ADDR=http://127.0.0.1:9112 \
    --rm --name profile-grpc registry.gitlab.com/target-digital-transformation/profile-be/master \
    /usr/local/bin/profile-serve profile grpc \
    --address=:9301 \
    --oidc='${env:// .ssoIssuer }' \
    --dsn='${env:// .profileDSN }' \
    --rabbitmq-addr='${env:// .rabbitMQAddr }' \
    --redis = '${env:// .redisAddr }' \
    --verbose
