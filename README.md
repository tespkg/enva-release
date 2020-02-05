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