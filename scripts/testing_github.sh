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

mysql --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} -e 'SELECT version();'
mysql --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} -e 'SELECT version();'

mysql --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} -e 'stop slave; reset slave all;'

mysql --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} -e 'reset master;'
mysql --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} -e 'reset master;'

mysql --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} -e "SHOW MASTER STATUS\G"
mysql --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} -e "SHOW MASTER STATUS\G"


# echo "bash main.sh --source-dsn ${ARGS_SOURCE_USER}/${ARGS_SOURCE_PASSWORD}@${ARGS_SOURCE_HOST}:${ARGS_SOURCE_PORT} --target-dsn ${ARGS_TARGET_USER}/${ARGS_TARGET_PASSWORD}@${ARGS_TARGET_HOST}:${ARGS_TARGET_PORT} --mysqlbinlog-connection-server-id "8889" --mysqlbinlog-stop-never"
# bash main.sh --source-dsn ${ARGS_SOURCE_USER}/${ARGS_SOURCE_PASSWORD}@${ARGS_SOURCE_HOST}:${ARGS_SOURCE_PORT} --target-dsn ${ARGS_TARGET_USER}/${ARGS_TARGET_PASSWORD}@${ARGS_TARGET_HOST}:${ARGS_TARGET_PORT} --mysqlbinlog-connection-server-id "8889" 


# echo "start $ARGS_TEST_NUM"
# function mysqlbinlog_sync0() {
#     python3.11 -u main.py --source-dsn "${ARGS_SOURCE_USER}/${ARGS_SOURCE_PASSWORD}@${ARGS_SOURCE_HOST}:${ARGS_SOURCE_PORT}" --target-dsn "${ARGS_TARGET_USER}/${ARGS_TARGET_PASSWORD}@${ARGS_TARGET_HOST}:${ARGS_TARGET_PORT}" --mysqlbinlog-stop-never
# }

# function mysqlbinlog_sync1() {
#     echo "build mysqlbinlog-statistics"
#     $HOME/.cargo/bin/cargo build
#     mysqlbinlog --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} --read-from-remote-source=BINLOG-DUMP-GTIDS --verify-binlog-checksum --connection-server-id=99999 --verbose --verbose --idempotent --force-read --print-table-metadata --stop-never mysql-bin.000001 2>/tmp/mysqlbinlog.2.log | ./target/debug/mysqlbinlog-statistics 2>/tmp/mysqlbinlog-statistics.2.log | mysql --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} >/dev/null 2>/tmp/mysql.2.log
# }

# function mysqlbinlog_sync2() {
#     mysqlbinlog --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} --read-from-remote-source=BINLOG-DUMP-GTIDS --verify-binlog-checksum --connection-server-id=99999 --verbose --verbose --idempotent --force-read --print-table-metadata --stop-never mysql-bin.000001 2>/tmp/mysqlbinlog.2.log | mysql --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} >/dev/null 2>/tmp/mysql.2.log
# }

# function mysqlbinlog_sync3() {
#     mysqlbinlog --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} --read-from-remote-source=BINLOG-DUMP-GTIDS --verify-binlog-checksum --connection-server-id=99999 --idempotent --force-read --print-table-metadata --stop-never mysql-bin.000001 2>/tmp/mysqlbinlog.2.log | mysql --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} >/dev/null 2>/tmp/mysql.2.log
# }

# function mysqlbinlog_sync4() {
#    mysqlbinlog --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} --read-from-remote-source=BINLOG-DUMP-GTIDS --verify-binlog-checksum --connection-server-id=99999 --idempotent --force-read --print-table-metadata --stop-never mysql-bin.000001 2>/dev/null | mysql --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} --verbose --verbose --verbose 2>/dev/null 1>/dev/null
# }

# # 设置无缓冲执行命令
# stdbuf -o0 command

# # 设置行缓冲执行命令
# stdbuf -oL command

# # 设置1M缓冲区执行命令
# stdbuf -o1M command
# stdbuf -o0 echo aabc | stdbuf -o0 grep aa | tail -n 111

# if [[ $ARGS_TEST_NUM == 'mysqlbinlog_sync0' ]]; then
#     mysqlbinlog_sync0 &
# elif [[ $ARGS_TEST_NUM == 'mysqlbinlog_sync4' ]]; then
#     mysqlbinlog_sync4 &
# elif [[ $ARGS_TEST_NUM == 'mysqlbinlog_sync1' ]]; then
#     mysqlbinlog_sync1 &
# elif [[ $ARGS_TEST_NUM == 'mysqlbinlog_sync2' ]]; then
#     mysqlbinlog_sync2 &
# elif [[ $ARGS_TEST_NUM == 'mysqlbinlog_sync3' ]]; then
#     mysqlbinlog_sync3 & 
# else
#     :
# fi
# MYSQLBINLOG_SYNC_PID=$!
# echo "$ARGS_TEST_NUM pid ${MYSQLBINLOG_SYNC_PID}"

sleep 5

# pstree -p $MYSQLBINLOG_SYNC_PID

echo "start load data to source database"
function sysbench_testing() {
    local testdb=$1
    mysql --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} -e "CREATE DATABASE IF NOT EXISTS ${testdb};"
    # for testname in oltp_insert oltp_delete oltp_update_index oltp_update_non_index oltp_write_only bulk_insert; do
    for testname in oltp_insert; do
        for action in cleanup prepare run; do
            echo "sysbench ${testdb}-${testname}-${action} start."
            sysbench /usr/share/sysbench/${testname}.lua --table-size=1000 --tables=10 --threads=100 --time=10 --mysql-db=${testdb} --mysql-host=${ARGS_SOURCE_HOST} --mysql-port=${ARGS_SOURCE_PORT} --mysql-user=${ARGS_SOURCE_USER} --mysql-password=${ARGS_SOURCE_PASSWORD} --db-driver=mysql $action >/dev/null
            echo "sysbench ${testdb}-${testname}-${action} done."
        done
    done
}


sysbench_testing "testdb_1"

mysqlbinlog -host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} --read-from-remote-source=BINLOG-DUMP-GTIDS --compression-algorithms=zstd --zstd-compression-level=3 --verify-binlog-checksum --to-last-log --connection-server-id=111 --verbose --verbose --idempotent --force-read --print-table-metadata mysql-bin.000001 | cargo run | mysql --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} -v -v

export ARGS_SOURCE_DSN="${ARGS_SOURCE_USER}/${ARGS_SOURCE_PASSWORD}@${ARGS_SOURCE_HOST}:${ARGS_SOURCE_PORT}"
export ARGS_TARGET_DSN="${ARGS_TARGET_USER}/${ARGS_TARGET_PASSWORD}@${ARGS_TARGET_HOST}:${ARGS_TARGET_PORT}"
export ARGS_DATABASES=testdb_1
curl -OLs https://raw.githubusercontent.com/shinhwagk/mysql-compare-tables/main/scripts/dbcompare.py
python /dbcompare.py