#!/bin/bash

mkdir -p build

date

[[ -f build/mysqlsync-testing ]] && sha1sum build/mysqlsync-testing

go build -o "build/mysqlsync-testing" ./src/*.go && sha1sum build/mysqlsync-testing