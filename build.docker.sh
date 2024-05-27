#!/bin/bash

docker build --platform linux/amd64 -t shinhwagk/mysqlsync:test-17 .

# docker rm -f mysqlsync-test
# docker run -d --name mysqlsync-test --network host -v /data/mysql-sync/:/opt/ -w /opt shinhwagk/mysqlsync:test-23 --config /opt/settings.py
# docker logs -f mysqlsync-test