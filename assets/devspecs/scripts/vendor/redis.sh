#!/bin/bash

docker run -d --network meera -p 6379:6379 --rm --name redis redis:alpine