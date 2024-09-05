#!/usr/bin/env bash

set -e

VERSION=`cat version`
docker pull --platform linux/amd64 docker.io/shinhwagk/mysql-sync:$VERSION
docker tag docker.io/shinhwagk/mysql-sync:$VERSION quay.io/shinhwagk/mysql-sync:$VERSION
docker tag docker.io/shinhwagk/mysql-sync:$VERSION quay.io/shinhwagk/mysql-sync:latest
docker push quay.io/shinhwagk/mysql-sync:$VERSION
docker push quay.io/shinhwagk/mysql-sync:latest