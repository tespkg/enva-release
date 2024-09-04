FROM registry.tespkg.in/library/enva:debian-buster-slim AS basis

FROM eclipse-temurin:11

RUN apt-get update && apt-get install -y --no-install-recommends \
	ca-certificates iputils-ping iproute2 \
	&& rm -rf /var/lib/apt/lists/*

COPY --from=basis /usr/local/envs/bin/enva /usr/local/envs/bin/enva
COPY --from=basis /docker-entrypoint.sh /docker-entrypoint.sh

ENV PATH="/usr/local/envs/bin:${PATH}"

ENTRYPOINT ["/docker-entrypoint.sh"]

