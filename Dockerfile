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

ENV PATH="/usr/local/envs/bin:${PATH}"

