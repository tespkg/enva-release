#!/bin/bash

docker run -d --network meera -p 9301:9301 \
    -e ENVS_HTTP_ADDR=http://host.docker.internal:9112 \
    --rm --name profile-graphql registry.gitlab.com/target-digital-transformation/profile-be/master \
    /usr/local/bin/profile-serve profile graphql \
    --address=:9301 \
    --oidc='${env:// .ssoIssuer }' \
    --dsn='${env:// .profileDSN }' \
    --rabbitmq-addr='${env:// .rabbitMQAddr }' \
    --redis='${env:// .redisAddr }' \
    --cors-hosts='${env:// .profileCORS }' \
    --ses-grpc='${env:// .sesGRPCAddr }' \
    --msg-pusher-grpc='${env:// .msgPusherGRPCAddr }' \
    --notification-addr='${env:// .notificationAddr }' \
    --debug-url-prefix=/graphql \
    --bypass-license \
    --verbose
