FROM registry.tespkg.in/library/enva:alpine AS basis

FROM nginx:alpine

RUN apk add --update ca-certificates openssl bash

COPY --from=basis /usr/local/envs/bin/enva /usr/local/envs/bin/enva
COPY --from=basis /docker-entrypoint.sh /docker-entrypoint.sh

ENV PATH="/usr/local/envs/bin:${PATH}"

ENTRYPOINT ["/docker-entrypoint.sh"]
