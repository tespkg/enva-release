#!/bin/bash

docker run -d -p 5555:5555 -p 5556:5556 --name app name/of/image /usr/local/bin/app --config ${envf:// .chapter01 } --dsn ${env:// .appdsn }