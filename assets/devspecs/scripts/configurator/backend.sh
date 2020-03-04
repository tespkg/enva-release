#!/bin/bash

docker run -d --network meera -p 9302:9302 -p 9303:9303 \
    -e ENVS_HTTP_ADDR=http://host.docker.internal:9112 \
    --rm --name configurator-be registry.gitlab.com/target-digital-transformation/subscription-store/master \
    /usr/local/bin/subscription-store serve \
    --grpc-address=:9302 \
    --address=:9303 \
    --oidc='${env:// .ssoIssuer }' \
    --dsn='${env:// .configuratorDSN }' \
    --sso-grpc='${env:// .ssoGRPCAddr }' \
    --ac-grpc='${env:// .acGRPCAddr }'  \
    --ses-grpc='${env:// .sesGRPCAddr }' \
    --profile-grpc='${env:// .profileGRPCAddr }' \
    --cors-hosts='${env:// .configuratorCORS }' \
    --rabbitmq-dsn='${env:// .rabbitMQAddr }' \
    --bypass-license \
    --verbose
