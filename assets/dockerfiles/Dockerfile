FROM debian:buster-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
	ca-certificates iputils-ping iproute2 curl \
	&& rm -rf /var/lib/apt/lists/*


# Download BI Connector to /mongosqld
WORKDIR /tmp
RUN curl https://info-mongodb-com.s3.amazonaws.com/mongodb-bi/v2/mongodb-bi-linux-x86_64-debian10-v2.14.0.tgz -o bi-connector.tgz && \
    tar -xvzf bi-connector.tgz && rm bi-connector.tgz && \
    mv /tmp/mongodb-bi-linux-x86_64-debian10-v2.14.0 /opt/mongosqld

ENV MONGODB_HOST mongodb
ENV MONGODB_PORT 27017
ENV MONGODB_USER root
ENV LISTEN_PORT 3307

CMD sh -c "/opt/mongosqld/bin/mongosqld --mongo-uri $MONGODB_HOST:$MONGODB_PORT --auth ---mongo-username $MONGODB_USER --mongo-password $MONGODB_PASSWORD -addr 0.0.0.0:$LISTEN_PORT"
