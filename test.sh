#!/bin/bash

set -exo pipefail

parse_connection_string() {
    local input=$1
    local username=$(echo $input | cut -d'/' -f1)
    local password_and_rest=$(echo $input | cut -d'/' -f2)
    local password=$(echo $password_and_rest | cut -d'@' -f1)
    local ip_and_port=$(echo $password_and_rest | cut -d'@' -f2)
    local ip=$(echo $ip_and_port | cut -d':' -f1)
    local port=$(echo $ip_and_port | cut -d':' -f2)

    echo "--user=$username --password=$password --host=$ip --port=$port"
}

# dnf install -y https://dev.mysql.com/get/mysql80-community-release-el9-5.noarch.rpm
# dnf install -y mysql-community-client

python3 generate_sql_statements.py | mysql $(parse_connection_string "root/example@db1:3306")

mysqlbinlog $(parse_connection_string "root/example@db1:3306") --read-from-remote-source=BINLOG-DUMP-GTIDS --compression-algorithms=zstd --zstd-compression-level=3 --verify-binlog-checksum --connection-server-id=11121 --verbose --verbose --idempotent --force-read --print-table-metadata --to-last-log mysql-bin.000001 | cargo run 2>temp/mysqlbinlog_statistics.err.log | mysql $(parse_connection_string "root/example@db2:3306") 2>temp/db2.err.log