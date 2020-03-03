FROM registry.tespkg.in/library/envs:alpine3.10 as basis

FROM nginx:alpine

RUN apk add --update ca-certificates openssl bash

COPY --from=basis /usr/local/envs/ /usr/local/envs/
COPY --from=basis /docker-entrypoint.sh /docker-entrypoint.sh

ENV PATH="/usr/local/envs/bin:${PATH}"

ENTRYPOINT ["/docker-entrypoint.sh"]
