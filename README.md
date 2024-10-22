# enva release

This repo is for TES environment agent (enva) release.

## How to upgrade / add enva to your project

Step 1: develop and test https://gitlab.com/target-digital-transformation/environment-store/
Step 2: tag e.g. `1.2.3`
Step 3: update https://github.com/tespkg/enva-release/blob/main/version with the tag `1.2.3`
Step 4: change the dockerfile of the repo, e.g.:

```dockerfile
FROM debian:bookworm-slim
ARG ENVA_VERSION=1.2.1
RUN apt-get update && \
  apt-get install -y --no-install-recommends ca-certificates libaio1 && \
  rm -rf /var/lib/apt/lists/* 
ADD --chmod=755 https://github.com/tespkg/enva-release/releases/download/${ENVA_VERSION}/enva_${ENVA_VERSION}_linux_amd64 /usr/local/bin/enva
COPY --from=gobuilder /go/bin/dex /usr/local/bin/
EXPOSE 5556
ENTRYPOINT ["/usr/local/bin/enva"]
```
