#!/bin/bash

VERSION=`cat version`

mkdir -p build

go build -o "build/mysqlsync-${VERSION}" ./src/*.go