FROM registry.tespkg.in/library/enva:alpine3.10 AS basis

FROM python:3.6.10-alpine3.10

RUN apk add --no-cache --update ca-certificates openssl bash

COPY --from=basis /usr/local/envs/bin/enva /usr/local/envs/bin/enva
COPY --from=basis /docker-entrypoint.sh /docker-entrypoint.sh

ENV PATH="/usr/local/envs/bin:${PATH}"

ENTRYPOINT ["/docker-entrypoint.sh"]

