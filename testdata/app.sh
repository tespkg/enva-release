#!/bin/bash

docker run -d --network host -e ENVS_HTTP_ADDR=http://localhost:9112 --name app name/of/image /usr/local/bin/app --config '${envf:// .chapter01 }' --dsn '${env:// .appdsn }'
