#!/bin/bash

set -e

declare ARGS_TEST_NUM=$1

declare ARGS_SOURCE_HOST="${ARGS_SOURCE_HOST}"
declare ARGS_SOURCE_PORT="${ARGS_SOURCE_PORT}"
declare ARGS_SOURCE_USER="${ARGS_SOURCE_USER}"
declare ARGS_SOURCE_PASSWORD="${ARGS_SOURCE_PASSWORD}"

declare ARGS_TARGET_HOST="${ARGS_TARGET_HOST}"
declare ARGS_TARGET_PORT="${ARGS_SOURCE_PORT}"
declare ARGS_TARGET_USER="${ARGS_SOURCE_USER}"
declare ARGS_TARGET_PASSWORD="${ARGS_TARGET_PASSWORD}"

function MYSQL_SOURCE_CLIENT {
    mysql --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} "$@"
}

function MYSQL_TARGET_CLIENT {
    mysql --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} "$@"
}

MYSQL_SOURCE_CLIENT -e "SELECT version();"
MYSQL_TARGET_CLIENT -e "SELECT version();"

# drop all non-sys databases;
echo "drop all non-sys databases;"
for db in `MYSQL_SOURCE_CLIENT -N -s -e "show databases" | grep -Ev '(information_schema|mysql|sys|performance_schema)'`; do
    MYSQL_SOURCE_CLIENT -N -s -e "drop database $db";
done

# drop all non-sys databases;
echo "drop all non-sys databases;"
for db in `MYSQL_TARGET_CLIENT -N -s -e "show databases" | grep -Ev '(information_schema|mysql|sys|performance_schema)'`; do
    MYSQL_TARGET_CLIENT -N -s -e "drop database $db";
done
 
MYSQL_SOURCE_CLIENT -e 'reset master;'
MYSQL_TARGET_CLIENT -e 'reset master;'

MYSQL_SOURCE_CLIENT -e "SHOW MASTER STATUS\G"
MYSQL_TARGET_CLIENT -e "SHOW MASTER STATUS\G"

echo "start load data to source database"
function sysbench_testing() {
    local testdb=$1
    MYSQL_SOURCE_CLIENT -e "CREATE DATABASE IF NOT EXISTS ${testdb};"
    # for testname in oltp_insert oltp_delete oltp_update_index oltp_update_non_index oltp_write_only bulk_insert; do
    for testname in oltp_insert; do
        for action in cleanup prepare run; do
            echo "sysbench ${testdb}-${testname}-${action} start."
            sysbench /usr/share/sysbench/${testname}.lua --table-size=100 --tables=4 --threads=10 --time=10 --mysql-db=${testdb} --mysql-host=${ARGS_SOURCE_HOST} --mysql-port=${ARGS_SOURCE_PORT} --mysql-user=${ARGS_SOURCE_USER} --mysql-password=${ARGS_SOURCE_PASSWORD} --db-driver=mysql $action >/dev/null
            echo "sysbench ${testdb}-${testname}-${action} done."
        done
    done
}

echo "load data complate."

sysbench_testing "testdb_1"

export RUST_BACKTRACE=1
mysqlbinlog --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} --read-from-remote-source=BINLOG-DUMP-GTIDS --compression-algorithms=zstd --zstd-compression-level=3 --verify-binlog-checksum --to-last-log --connection-server-id=111 --verbose --verbose --idempotent --force-read --print-table-metadata mysql-bin.000001 | $HOME/.cargo/bin/cargo run | MYSQL_TARGET_CLIENT --verbose --verbose

export ARGS_SOURCE_DSN="${ARGS_SOURCE_USER}/${ARGS_SOURCE_PASSWORD}@${ARGS_SOURCE_HOST}:${ARGS_SOURCE_PORT}"
export ARGS_TARGET_DSN="${ARGS_TARGET_USER}/${ARGS_TARGET_PASSWORD}@${ARGS_TARGET_HOST}:${ARGS_TARGET_PORT}"
export ARGS_DATABASES=testdb_1
(mkdir /tmp/compare && cd /tmp/compare && python3.11 /workspaces/mysqlbinlog-sync/scripts/dbcompare.py)

find /tmp/compare -type f -name '*.diff.log' 