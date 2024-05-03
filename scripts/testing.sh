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

mysql --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} -e 'SELECT version();'


# echo "build mysqlbinlog-statistics"
# $HOME/.cargo/bin/cargo build

echo "start mysqlbinlog-sync"
function mysqlbinlog_sync0() {
    python3.11 -u main.py --source-dsn "${ARGS_SOURCE_USER}/${ARGS_SOURCE_PASSWORD}@${ARGS_SOURCE_HOST}:${ARGS_SOURCE_PORT}" --target-dsn "${ARGS_TARGET_USER}/${ARGS_TARGET_PASSWORD}@${ARGS_TARGET_HOST}:${ARGS_TARGET_PORT}" --mysqlbinlog-stop-never
}

function mysqlbinlog_sync1() {
    mysqlbinlog --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} --read-from-remote-source=BINLOG-DUMP-GTIDS --verify-binlog-checksum --connection-server-id=99999 --verbose --verbose --idempotent --force-read --print-table-metadata --stop-never mysql-bin.000001 2>/tmp/mysqlbinlog.2.log | ./target/debug/mysqlbinlog-statistics 2>/tmp/mysqlbinlog-statistics.2.log | mysql --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} >/dev/null 2>/tmp/mysql.2.log
}

function mysqlbinlog_sync2() {
    mysqlbinlog --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} --read-from-remote-source=BINLOG-DUMP-GTIDS --verify-binlog-checksum --connection-server-id=99999 --verbose --verbose --idempotent --force-read --print-table-metadata --stop-never mysql-bin.000001 2>/tmp/mysqlbinlog.2.log | mysql --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} >/dev/null 2>/tmp/mysql.2.log
}

function mysqlbinlog_sync3() {
    mysqlbinlog --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} --read-from-remote-source=BINLOG-DUMP-GTIDS --verify-binlog-checksum --connection-server-id=99999 --idempotent --force-read --print-table-metadata --stop-never mysql-bin.000001 2>/tmp/mysqlbinlog.2.log | mysql --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} >/dev/null 2>/tmp/mysql.2.log
}

if [[ $ARGS_TEST_NUM == 'mysqlbinlog_sync0' ]]; then
    mysqlbinlog_sync0 &
elif [[ $ARGS_TEST_NUM == 'mysqlbinlog_sync1' ]]; then
    mysqlbinlog_sync1 &
elif [[ $ARGS_TEST_NUM == 'mysqlbinlog_sync2' ]]; then
    mysqlbinlog_sync2 &
elif [[ $ARGS_TEST_NUM == 'mysqlbinlog_sync3' ]]; then
    mysqlbinlog_sync3 & 
else
    :
fi
MYSQLBINLOG_SYNC_PID=$!
echo "mysqlbinlog_sync3 pid ${MYSQLBINLOG_SYNC_PID}"

echo "start load data to source database"
mysql --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} -e 'CREATE DATABASE IF NOT EXISTS testdb;'
# for testname in oltp_insert oltp_delete oltp_update_index oltp_update_non_index oltp_write_only bulk_insert; do
for testname in oltp_insert; do
    for action in cleanup prepare run cleanup; do
        echo "sysbench ${testname}-${action}"
        sysbench /usr/share/sysbench/${testname}.lua --table-size=100000 --tables=10 --threads=100 --time=10 --mysql-db=testdb --mysql-host=${ARGS_SOURCE_HOST} --mysql-port=${ARGS_SOURCE_PORT} --mysql-user=${ARGS_SOURCE_USER} --mysql-password=${ARGS_SOURCE_PASSWORD} --db-driver=mysql $action
    done
done

start_ts=`date +%s`
for i in `seq 1 600`; do
    SOURCE_GTID=$(mysql --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} -e 'show master status\G' 2>/dev/null| grep Executed_Gtid_Set)
    TARGET_GTID=$(mysql --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} -e 'show master status\G' 2>/dev/null| grep Executed_Gtid_Set)
    if [[ "$SOURCE_GTID" == "$TARGET_GTID" ]]; then
        break
    fi
    echo "source gtid ${SOURCE_GTID}"
    echo "target gtid ${TARGET_GTID}"
    sleep 1
done

echo "kill sync ${MYSQLBINLOG_SYNC_PID}"
kill $MYSQLBINLOG_SYNC_PID

end_ts=`date +%s`
echo "sync $((end_ts-start_ts))s"
