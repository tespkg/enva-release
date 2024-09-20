FROM golang:1.22-alpine AS builder

RUN apk --no-cache add --update alpine-sdk bash

COPY . /build
WORKDIR /build
RUN make build

FROM alpine

RUN apk add --update ca-certificates openssl busybox-extras bash

COPY --from=builder /build/bin/envs /usr/local/envs/bin/envs
COPY --from=builder /build/static /usr/local/envs/static
COPY --from=builder /build/docker-entrypoint.sh /docker-entrypoint.sh

ENV PATH="/usr/local/envs/bin:${PATH}"

ENTRYPOINT ["/docker-entrypoint.sh"]
