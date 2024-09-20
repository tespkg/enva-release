FROM registry.tespkg.in/library/enva:alpine AS basis

FROM python:3.6.10-alpine

RUN apk add --no-cache --update ca-certificates openssl bash

COPY --from=basis /usr/local/envs/bin/enva /usr/local/envs/bin/enva
COPY --from=basis /docker-entrypoint.sh /docker-entrypoint.sh

ENV PATH="/usr/local/envs/bin:${PATH}"

ENTRYPOINT ["/docker-entrypoint.sh"]

