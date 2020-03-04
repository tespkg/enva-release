#!/bin/bash

docker run -d --network meera -p 5432:5432 -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=password -e PGDATA=/var/lib/postgresql/data --rm --name postgres postgres:11.1-alpine