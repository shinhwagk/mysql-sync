#!/usr/bin/env bash

set -o pipefail

mysqlbinlog | mysqlbinlog_statistics | mysql

cmd1 | cmd2 | cmd3 

status=$?

echo "Exit status: $status"
if [ $status -eq 0 ]; then
    echo "All commands executed successfully"
else
    echo "cmd2 failed after 5 seconds"
fi
