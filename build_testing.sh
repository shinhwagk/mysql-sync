#!/bin/bash

go build -ldflags="-s -w" -o /tmp/mysqlsync ./src/*.go