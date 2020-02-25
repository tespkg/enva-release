# environtment store

0. start all db instances
1. start env store (register db instances)
2. start e.g. sso, retrieve db connection etc, publish sso issuer URL
3. start e.g. access-control..
...
10. start front-end workspace, retrieve service endpoints, publish workspace url
...

POC: run everything inside one docker, consider:
1. Consul as env store, 
2. Wrap every service or front-end as start script or 
3. Front-end start should use a Go file server

## goal
1. service start command line args, options and configs in the yaml files will not need to change at most time.
1. move changeable command line args, options and configs from yaml file to config store.

## key register & setup flow
1. (DevOps) Start envs + state
1. (Auto) envs scan application/service specs
1. (Auto) Required keys scanned
1. (Auto) Swagger.json ready
1. (DevOps) Get swagger.json
1. (DevOps) PUT /keys {key1: val1, ..., keyN: valN}
1. (DevOps) Start service
1. (Dev) Upload new application spec, if there is a new application/service developed

## convention
1. Required key `{env:// .key }`
1. Required file key `{envf:// .keyf }`
1. Optional key `{envo:// .key }`
1. Optional file key `{envof:// .keyf }`
1. Filename annotation `{envfn: filename}` 
1. Allowed key name pattern `{env(f|o|of)?:// *\.([_a-zA-Z][_a-zA-Z0-9]*) *}`
1. Allowed filename annotation `{envfn: *([-_a-zA-Z0-9]*) *}`

## TODO
- [x] enva start application/service
- [x] Support envf
- [x] Scan application/service spec
- [x] Render application/service spec from env store
- [ ] Implement keys query on store level
- [ ] Implement GET, PUT REST APIs for keys
- [ ] Implement Register REST APIS for new application/spec
- [ ] Refactor enva to use envs instead of using naked underlying etcd/consul
- [ ] An addons way to extend the pre-configuration for service startup, e.g, create database if not exist etc.
- [ ] Wrap images to include `enva`, `s4`(simple static site service) binary
- [ ] Local app specs for dev purpose
- [ ] Serve front end with `s4`
- [ ] Support key watch & restart 
- [ ] Kubernetes operator...
- [ ] env store on k8s, istio