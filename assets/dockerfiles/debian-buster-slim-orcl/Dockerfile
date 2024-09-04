FROM registry.tespkg.in/library/enva:debian-buster-slim AS basis

FROM debian:buster-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
	ca-certificates iputils-ping iproute2 unzip libaio1 \
	&& rm -rf /var/lib/apt/lists/*

COPY . /build
WORKDIR /build

RUN /build/instantclient185.sh
RUN rm -rf /build

COPY --from=basis /usr/local/envs/bin/enva /usr/local/envs/bin/enva
COPY --from=basis /docker-entrypoint.sh /docker-entrypoint.sh

ENV PATH="/usr/local/envs/bin:${PATH}"
ENV PKG_CONFIG_PATH=/opt/oracle
ENV LD_LIBRARY_PATH=/opt/oracle/instantclient_18_5

ENTRYPOINT ["/docker-entrypoint.sh"]

