FROM golang:1.22-bullseye AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
	&& rm -rf /var/lib/apt/lists/*

COPY . /build
WORKDIR /build
RUN make build

FROM debian:stable-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
	ca-certificates\
	&& rm -rf /var/lib/apt/lists/*

COPY --from=builder /build/bin/enva /usr/local/envs/bin/enva
COPY --from=builder /build/docker-entrypoint-enva.sh /docker-entrypoint.sh

ENV PATH="/usr/local/envs/bin:${PATH}"

ENTRYPOINT ["/docker-entrypoint.sh"]

