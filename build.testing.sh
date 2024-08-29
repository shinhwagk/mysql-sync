#!/bin/bash

mkdir -p build

go build -o "build/mysqlsync-testing" ./src/*.go