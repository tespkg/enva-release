#!/bin/bash

docker run -d --network meera -p 8080:8080 \
    -e ENVS_HTTP_ADDR=http://host.docker.internal:9112 \
    -e VUE_APP_API_ENDPOINT='${env:// .acRESTAddr }' \
    -e VUE_APP_CLIENT_ID=access-control \
    -e VUE_APP_CLIENT_SECRET=DzXZxyDObSpsnR7qLqQ4p1LEVoIiE49e \
    -e VUE_APP_STATE=acconsole \
    -e VUE_APP_TOKEN_URL='${env:// .oidcIssuerTokenURL }' \
    -e VUE_APP_AUTH_URL='${env:// .oidcIssuerAuthURL }' \
    -e VUE_APP_REDIRECT_URL='${env:// .acConsoleAddr }'/oauth2 \
    --rm --name acconsole registry.gitlab.com/target-digital-transformation/access-control/console/master