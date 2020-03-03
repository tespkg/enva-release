FROM golang:1.13-alpine3.10 as builder

RUN apk --no-cache add --update alpine-sdk bash

COPY . /build
WORKDIR /build
RUN make build

FROM alpine:3.10

RUN apk add --update ca-certificates openssl busybox-extras bash

COPY --from=builder /build/bin/enva /usr/local/envs/bin/enva
COPY --from=builder /build/docker-entrypoint.sh /docker-entrypoint.sh

ENV PATH="/usr/local/envs/bin:${PATH}"

ENTRYPOINT ["/docker-entrypoint.sh"]
