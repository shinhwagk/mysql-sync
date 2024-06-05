#!/bin/bash

version=`cat version`
docker build -t shinhwagk/mysqlsync:${version} -f Dockerfile.go .

docker push shinhwagk/mysqlsync:${version}