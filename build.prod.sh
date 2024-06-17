#!/bin/bash

VERSION=`cat version`
docker build -f Dockerfile.go -t shinhwagk/mysql-sync:${VERSION} .
docker tag shinhwagk/mysql-sync:${VERSION} shinhwagk/mysql-sync:latest

docker push shinhwagk/mysql-sync:${VERSION}
docker push shinhwagk/mysql-sync:latest
