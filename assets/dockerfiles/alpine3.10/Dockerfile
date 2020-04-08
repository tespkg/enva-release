FROM registry.tespkg.in/library/enva:alpine3.10 as basis

FROM alpine:3.10

RUN apk add --update ca-certificates openssl bash

COPY --from=basis /usr/local/envs/bin/enva /usr/local/envs/bin/enva
COPY --from=basis /docker-entrypoint.sh /docker-entrypoint.sh

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

