#!/usr/bin/env bash

git clone --depth=1 https://github.com/shinhwagk/mysqlbinlog-sync app

python -u /app/main1.py "$@"