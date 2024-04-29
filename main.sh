#!/usr/bin/env bash

set -o pipefail

mysqlbinlog | mysqlbinlog_statistics | mysql

cmd1 | cmd2 | cmd3

declare MYSQLBINLOG_SERVER_ID=$((RANDOM + 1000000000))

mysqlbinlog --host=db1 --port=3306 --user=root --password=example \
  --read-from-remote-source=BINLOG-DUMP-GTIDS \
  --compression-algorithms=zstd --zstd-compression-level=3 \
  --verify-binlog-checksum \
  --to-last-log \
  --connection-server-id=11121 --verbose --verbose --idempotent --force-read --print-table-metadata mysql-bin.000001