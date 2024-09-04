FROM registry.tespkg.in/library/enva:alpine3.10 AS basis

FROM node:20-alpine3.18

RUN apk add --update ca-certificates openssl bash

COPY --from=basis /usr/local/envs/bin/enva /usr/local/envs/bin/enva
COPY --from=basis /docker-entrypoint.sh /docker-entrypoint.sh

ENV PATH="/usr/local/envs/bin:${PATH}"

ENTRYPOINT ["/docker-entrypoint.sh"]
