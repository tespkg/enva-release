#!/bin/bash

docker run --network host -p 50051:50051 \
    -e ENVS_HTTP_ADDR=http://127.0.0.1:9112 \
    --rm --name profile-grpc registry.gitlab.com/target-digital-transformation/profile-be/master \
    /usr/local/bin/profile-serve profile grpc \
    --address=:50051 \
    --oidc='${env:// .ssoIssuer }' \
    --dsn='${env:// .profileDSN }' \
    --rabbitmq-addr='${env:// .rabbitMQAddr }' \
    --sso-dex-grpc-addr='${env:// .ssoGRPCAddr }' \
    --verbose
