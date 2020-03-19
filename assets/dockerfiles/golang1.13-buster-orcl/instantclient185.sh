#!/bin/bash

set -e
set -o errexit
set -o pipefail
shopt -s nullglob

mkdir -p /opt/oracle
cp instantclient-basiclite-linux.x64-18.5.0.0.0dbru.zip /opt/oracle/
cp instantclient-sdk-linux.x64-18.5.0.0.0dbru.zip /opt/oracle/
cp instantclient-sqlplus-linux.x64-18.5.0.0.0dbru.zip /opt/oracle/
cp oci8.pc /opt/oracle/
cd /opt/oracle
unzip instantclient-basiclite-linux.x64-18.5.0.0.0dbru.zip
unzip instantclient-sdk-linux.x64-18.5.0.0.0dbru.zip
unzip instantclient-sqlplus-linux.x64-18.5.0.0.0dbru.zip
rm -rf instantclient-basiclite-linux.x64-18.5.0.0.0dbru.zip instantclient-sdk-linux.x64-18.5.0.0.0dbru.zip instantclient-sqlplus-linux.x64-18.5.0.0.0dbru.zip
cd /opt/oracle/instantclient_18_5
echo /opt/oracle/instantclient_18_5 > /etc/ld.so.conf.d/oracle-instantclient.conf
ldconfig
