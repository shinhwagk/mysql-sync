#!/bin/bash

build() {
  go build -o /tmp/mysqlsync ./src/*.go
#   go build -ldflags="-s -w" -o /tmp/mysqlsync ./src/*.go

}

if [[ $1 == "-w" ]]; then
    while true; do
        build && date
        sleep 1
    done
else
    build
fi
