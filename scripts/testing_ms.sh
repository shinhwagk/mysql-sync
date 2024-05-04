#!/bin/bash

set -e

declare ARGS_SOURCE_HOST="${ARGS_SOURCE_HOST:-db1}"
declare ARGS_SOURCE_PORT="${ARGS_SOURCE_PORT:-3306}"
declare ARGS_SOURCE_USER="${ARGS_SOURCE_USER:-root}"
declare ARGS_SOURCE_PASSWORD="${ARGS_SOURCE_PASSWORD:-example}"

declare ARGS_TARGET_HOST="${ARGS_TARGET_HOST:-db2}"
declare ARGS_TARGET_PORT="${ARGS_SOURCE_PORT:-3306}"
declare ARGS_TARGET_USER="${ARGS_SOURCE_USER:-root}"
declare ARGS_TARGET_PASSWORD="${ARGS_TARGET_PASSWORD:-example}"

mysql --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} -e 'SELECT version();'
mysql --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} -e 'SELECT version();'

mysql --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} -e 'stop slave; reset slave all; reset master;'
mysql --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} -e 'stop slave; reset slave all; reset master;'

mysql --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} -e "CHANGE REPLICATION SOURCE TO SOURCE_HOST=\"${ARGS_SOURCE_HOST}\",SOURCE_PORT=${ARGS_SOURCE_PORT},SOURCE_USER=\"${ARGS_SOURCE_USER}\",SOURCE_PASSWORD=\"${ARGS_SOURCE_PASSWORD}\",SOURCE_AUTO_POSITION=1;"
mysql --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} -e "START SLAVE; SHOW SLAVE STATUS\G"



echo "start load data to source database"
function sysbench_testing() {
    local testdb=$1
    mysql --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} -e "CREATE DATABASE IF NOT EXISTS ${testdb};"
    # for testname in oltp_insert oltp_delete oltp_update_index oltp_update_non_index oltp_write_only bulk_insert; do
    for testname in oltp_insert oltp_delete oltp_update_index; do
        for action in cleanup prepare run cleanup; do
            echo "sysbench ${testdb}-${testname}-${action} start."
            sysbench /usr/share/sysbench/${testname}.lua --table-size=100000 --tables=10 --threads=100 --time=10 --mysql-db=${testdb} --mysql-host=${ARGS_SOURCE_HOST} --mysql-port=${ARGS_SOURCE_PORT} --mysql-user=${ARGS_SOURCE_USER} --mysql-password=${ARGS_SOURCE_PASSWORD} --db-driver=mysql $action >/dev/null
            echo "sysbench ${testdb}-${testname}-${action} done."
        done
    done
}

for dbid in `seq 1 3`; do
    sysbench_testing "testdb_${dbid}" &
done

for i in `seq 1 600`; do
    mysql --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} -e 'show master status\G'
    mysql --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} -e 'show master status\G'
    sleep 1
done

wait
# kill $MYSQLBINLOG_SYNC_PID
