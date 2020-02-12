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
2. move changable command line args, options and configs from yaml file to config store.

## key register & setup flow
1. enva register the service's endpoint(internal, external) key name to store.
2. ops set values for those registed key manually in store.
3. enva consume the value with the registered key name and feed them to consumer service.
4. envs migration tool provide a set of pre-defined key name of vender's endpoint to store.
5. devops setup vendor's endpoint value under the pre-defined key into store manually.

ps:
- service's public endpoint which can be access from outside, e.g, oidc-issuer endpoint.
- service's internal endpoint, e.g, grpc endpoint.

## convention
1. key name convention for the endpoint: <servicename>-<protocol>, e.g, sso-grpc, ac-http, ac-grpc configurator-graphql...
2. each service will have it's own dsn key
3. a better token for env, envf schema in command line: {env://abc}, {envf://abc}
4. token in the files would be: %% .ENV_<project>_<keyname> %%, use `%%` instead of `{{` to avoid conflict in html or js code.

## envf explanation
1. command line: `envf://sso-config`
2. store: `sso-config: /path/to/file.yaml=ENV_content_of_file`
3. store: `content_of_file: blablabla....`
