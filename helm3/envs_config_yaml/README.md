# Accesscontrol-be

## Introduction

Workspace-be is an identity service that uses OpenID Connect to drive authentication for other apps.
Sso acts as a portal to other identity providers through "connectors." This lets sso defer authentication to LDAP servers, SAML providers, or established identity providers like GitHub, Google, and Active Directory. Clients write their authentication logic once to talk to sso, then sso handles the protocols for a given backend.

## Dependencies
```yaml
  - name: sso-be
    version: 3.0.6-alpha
  - name: accesscontrol-be
    version: 3.0.0
  - name: postgres
    version: 10.9
```

## [Release Note](https://gitlab.com/target-digital-transformation/access-control/-/tags)

## Upgrading an existing Release to a new major version

> Supported upgrade paths: 
``` 
2.0.0 (or older) -> 2.5.1
2.5.1            -> 3.0.0
3.0.0            -> 3.1.0 (or newer)
```

there maybe an incompatible breaking change needing manual actions.
## migration guide (there is an incompatible breaking change needing manual actions):

### To 3.0.0

This version causes a change in the Redis Master StatefulSet definition, so the command helm upgrade would not work out of the box. As an alternative, one of the following could be done:

  - Recommended: Create a clone of the Redis Master PVC (for example, using projects like [this one](https://github.com/edseymour/pvc-transfer)). Then launch a fresh release reusing this cloned PVC.

## Installing the Chart
```sh
nameSpace=devops-meeraspace
workspaceHost=ops.meeraspace.com
nameSpacePrefix=`echo $nameSpace | awk -F "-" '{print $1}'` && echo " nameSpacePrefix is $nameSpacePrefix "  ## return  devops
nameSpaceMid=`echo $nameSpace | awk -F "-" '{print $2}'` && echo " nameSpaceMid is $nameSpaceMid "  ## return  meerasapce
hostPrefix=`echo $workspaceHost | awk -F "." '{print $1}'` && echo " hostPrefix is $hostPrefix "  ## return  ops
baseUrl=`echo ${workspaceHost#$hostPrefix.}` && echo " baseUrl is $baseUrl "   ## return meeraspace.com
```
To install the chart with the release name `release-name` & the global.dev `yours` :
```bash
helm3 install release-name  --set global.env=$yours,global.mode.standard=false,global.namespace.prefix=$nameSpacePrefix,global.namespace.mid=$nameSpaceMid,global.host.prefix=$hostPrefix,global.host.baseUrl=$baseUrl \
-n $nameSpace-system meeraspace-stable/${chartName} --version $version
```

## Parameters
The following table lists the configurable parameters of the workspace-be chart and their default values.

| Parameter                                     | Description                                                                                                                                         | Default                                                 |
|-----------------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------------------------|
| `global.env`                                  | Global Docker registry secret names as an array                                                                                                     | `dev` |
| `global.mode.standard `                       | Global Docker registry secret names as an array                                                                                                | `true` |
| `image.repository`                            |                                                                                                            | registry.gitlab.com/target-digital-transformation/meera-ws               |
| `image.tag`                                   | Redis Image tag                                                                                                                                     | `{TAG_NAME}`                                            |
| `image.config.DATABASE`                       | Database name                                                                                                                                     | ws                                            |
| `istio.virtualservice.enabled`                | Use virtualservice                                                                                                                                  | `true`                                                  |
| `replicaCount      `                          | Number of pods                                                                                                                                    | `1`                                                     |