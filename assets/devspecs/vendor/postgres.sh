#!/bin/bash

docker run -d --network host -p 5432:5432 -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=password -e PGDATA=/var/lib/postgresql/data --name postgres postgres:11.1-alpine