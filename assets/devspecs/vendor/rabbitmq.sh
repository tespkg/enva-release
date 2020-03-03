#!/bin/bash

docker run -d --network host -p 5672:5672 -p 15672:15672 -p 61613:61613 --name rabbitstomp itzg/rabbitmq-stomp