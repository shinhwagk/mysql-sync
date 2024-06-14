#!/bin/bash

VERSION=`cat version`
docker build -f Dockerfile.go -t shinhwagk/mysqlsync:${VERSION} .
docker tag shinhwagk/mysqlsync:${VERSION} shinhwagk/mysqlsync:latest

docker push shinhwagk/mysqlsync:${VERSION}
docker push shinhwagk/mysqlsync:latest
