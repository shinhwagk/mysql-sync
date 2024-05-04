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

sleep 5

start_ts=`date +%s`
target_gtid_num=0
for i in `seq 1 600`; do
    SOURCE_GTID=$(mysql --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} -e 'show master status\G' 2>/dev/null | grep Executed_Gtid_Set)
    TARGET_GTID=$(mysql --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} -e 'show master status\G' 2>/dev/null | grep Executed_Gtid_Set)
    
    echo "source gtid ${SOURCE_GTID}"
    echo "target gtid ${TARGET_GTID}"

    curr_target_gtid_num="${TARGET_GTID#*:1-}"
    echo "gtid sync $(( (curr_target_gtid_num - target_gtid_num) / 10 ))/s"
    target_gtid_num=$curr_target_gtid_num

    echo "sysbench process number: `ps -ef | grep sysbench | grep -v grep | wc -l`"
    # mysql --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} -e 'show processlist;'

    if [[ "$SOURCE_GTID" == "$TARGET_GTID" && `ps -ef | grep sysbench | grep -v grep | wc -l` == 0 ]]; then
        break;
    fi

    sleep 10
done

end_ts=`date +%s`
echo "sync success $((end_ts-start_ts))s"
