#!/bin/bash

docker run -d --network host -p 8500:8500 -p 8502:8502 -p 8600:8600 --network host --name consul consul consul agent -client=0.0.0.0 -dev