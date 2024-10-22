# enva release

This repo is for environment agent (enva) binary release only. It contains only the github workflow which 

1. clones the gitlab project
2. build only the `cmd/enva` module (CGO_ENABLED=0)
3. publish the `enva` binary for linux amd64 and arm64.

The binary can then be used directly in each projects Dockerfile, thus skipping building separate base images like tespkg.in/library/debian, tespkg.in/library/ubuntu etc.

## How to upgrade / add enva to your project

1. develop and test https://gitlab.com/target-digital-transformation/environment-store/
2. tag e.g. `1.2.3`
3. update https://github.com/tespkg/enva-release/blob/main/version with the tag `1.2.3`
4. change the dockerfile of the repo, e.g.:

```dockerfile
FROM debian:bookworm-slim
ARG ENVA_VERSION=1.2.3
RUN apt-get update && \
  apt-get install -y --no-install-recommends ca-certificates libaio1 && \
  rm -rf /var/lib/apt/lists/* 
ADD --chmod=755 https://github.com/tespkg/enva-release/releases/download/${ENVA_VERSION}/enva_${ENVA_VERSION}_linux_amd64 /usr/local/bin/enva
COPY --from=gobuilder /go/bin/dex /usr/local/bin/
EXPOSE 5556
ENTRYPOINT ["/usr/local/bin/enva"]
```

Owner for each project needs to decide when and how to upgrade the `enva` binary, e.g. via

1. dockerfile
2. repo variable
3. org vairable
