#!/bin/bash

VERSION=`cat version`

go mod tidy
echo "building linux/amd64 binaries"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags="-s -w" -o "build/mysqlsync-amd64-${VERSION}" ./src/*.go
echo "building linux/arm64 binaries"
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -a -ldflags="-s -w" -o "build/mysqlsync-arm64-${VERSION}" ./src/*.go