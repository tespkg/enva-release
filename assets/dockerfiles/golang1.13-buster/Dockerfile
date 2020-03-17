FROM registry.tespkg.in/library/debian:buster-slim as basis

FROM golang:1.13-buster

RUN apt-get update && apt-get install -y --no-install-recommends \
	ca-certificates iputils-ping iproute2 \
	&& rm -rf /var/lib/apt/lists/*

COPY --from=basis /usr/local/envs/bin/enva /usr/local/envs/bin/enva
COPY --from=basis /docker-entrypoint.sh /docker-entrypoint.sh

ENV PATH="/usr/local/envs/bin:${PATH}"

ENTRYPOINT ["/docker-entrypoint.sh"]


