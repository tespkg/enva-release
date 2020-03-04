#!/bin/bash

docker run -d --network meera -p 5672:5672 -p 15672:15672 -p 61613:61613 --rm --name rabbitstomp itzg/rabbitmq-stomp