#!/bin/bash

VERSION=`cat version`

function build_image() {
    plat=$1
    echo "build platform: ${plat}"
    docker build -f Dockerfile.go --platform $plat -t shinhwagk/mysql-sync:${VERSION} .
    docker tag shinhwagk/mysql-sync:${VERSION} shinhwagk/mysql-sync:latest

    docker push shinhwagk/mysql-sync:${VERSION}
    docker push shinhwagk/mysql-sync:latest
}

for plat in linux/amd64 linux/arm64; do
    build_image $plat
done