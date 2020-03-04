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

## Goal
1. Service start command line args, options and configs in the yaml files(equal to `application/service spec` in future context) will not need to change at most time.
1. Move changeable command line args, options and configs from application/service spec to env store.

## Key register & setup flow
1. (DevOps) Start envs + state
1. (Auto) envs scan application/service specs
1. (Auto) Required keys scanned
1. (Auto) Swagger.json ready
1. (DevOps) Get swagger.json
1. (DevOps) PUT /keys {key1: val1, ..., keyN: valN}
1. (DevOps) Start service
1. (Dev) Upload new application/service spec to envs, if there is a new application/service developed

## Conventions
1. Required key `${env:// .key }`
1. Required file key `${envf:// .keyf }`
1. Allowed key name pattern `\${env(f)?:// *\.([_a-zA-Z][_a-zA-Z0-9]*) *(\| *default ([./_a-zA-Z0-9]*))? *}`

## TODO
- [x] enva start application/service
- [x] Support envf
- [x] Scan application/service spec
- [x] Render application/service spec from env store
- [x] Implement query on store level for keys
- [x] Implement GET, PUT REST APIs for keys
- [x] Implement Register REST APIS for new application/service spec
- [x] Serve swagger.json
- [x] Refactor enva to use envs instead of using naked underlying etcd/consul
- [x] Intercept os ENV vars and render os env files
- [x] Wrap sso, ac, profile, configurator images to include `enva`~~, `s4`(simple static site service) binary~~
- [x] ~~Serve front end with s4~~, [continue to use nginx to serve front end](#Keep-using-nginx-as-the-frontend-site-server)
- [x] Export & Import Key values APIs 
- [x] Local app specs for dev purpose
- [x] Support env/envf with default value, if given default value and the key doesn't existed before, publish to underlying store
- [ ] enva publish kvs to envs
- [ ] Support key watch & restart 
- [ ] Replace nginx with our own `simple static site service(s4)`
- [ ] An extensive way to extend the pre-configuration for service startup, e.g, create database if not exist etc.
- [ ] ~~Migrate specs at startup~~
- [ ] Kubernetes operator...
- [ ] API for starting service
- [ ] env store on k8s, istio

## FAQs

### Keep using nginx as the frontend site server

1. It will require only very little changes to adopt to `envs` management flow
1. It's a full featured & lightweight web server
1. Will replace nginx with our own site service `s4` eventually
