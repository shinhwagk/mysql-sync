#!/usr/bin/env bash

git clone --depth=1 https://github.com/shinhwagk/mysql-sync /app >/dev/null 2>&1

python -u /app/main.py "$@"