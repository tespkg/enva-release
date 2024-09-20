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

# Create minimal nsswitch.conf file to prioritize the usage of /etc/hosts over DNS queries.
# This resolves the conflict between:
# * fluxd using netgo for static compilation. netgo reads nsswitch.conf to mimic glibc,
#   defaulting to prioritize DNS queries over /etc/hosts if nsswitch.conf is missing:
#   https://github.com/golang/go/issues/22846
# * Alpine not including a nsswitch.conf file. Since Alpine doesn't use glibc
#   (it uses musl), maintainers argue that the need of nsswitch.conf is a Go bug:
#   https://github.com/gliderlabs/docker-alpine/issues/367#issuecomment-354316460
RUN [ ! -e /etc/nsswitch.conf ] && echo 'hosts: files dns' > /etc/nsswitch.conf

ENV PATH="/usr/local/envs/bin:${PATH}"

ENTRYPOINT ["/docker-entrypoint.sh"]
