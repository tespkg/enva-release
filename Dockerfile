FROM golang:1.13-buster as builder

RUN apt-get update && apt-get install -y --no-install-recommends \
	&& rm -rf /var/lib/apt/lists/*

COPY . /build
WORKDIR /build
RUN make build

FROM debian:buster-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
	ca-certificates\
	&& rm -rf /var/lib/apt/lists/*

COPY --from=builder /build/bin/enva /usr/local/envs/bin/enva
COPY --from=builder /build/bin/envi /usr/local/envs/bin/envi
COPY --from=builder /build/bin/s4 /usr/local/envs/bin/s4
COPY --from=builder /build/docker-entrypoint.sh /docker-entrypoint.sh

ENV PATH="/usr/local/envs/bin:${PATH}"

ENTRYPOINT ["/docker-entrypoint.sh"]
