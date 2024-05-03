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

mysql --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} -e 'reset master;'
mysql --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} -e 'reset master;'

mysql --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} -e "CHANGE REPLICATION SOURCE TO SOURCE_HOST='${ARGS_SOURCE_HOST}',SOURCE_PORT=${ARGS_SOURCE_PORT},SOURCE_USER='${ARGS_SOURCE_USER}',SOURCE_PASSWORD='${ARGS_SOURCE_PASSWORD}',SOURCE_AUTO_POSITION=1;"
mysql --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} -e "START SLAVE;"

echo "start load data to source database"
mysql --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} -e 'CREATE DATABASE IF NOT EXISTS testdb;'
# for testname in oltp_insert oltp_delete oltp_update_index oltp_update_non_index oltp_write_only bulk_insert; do
for testname in oltp_insert; do
    for action in cleanup prepare run cleanup; do
        echo "sysbench ${testname}-${action}"
        sysbench /usr/share/sysbench/${testname}.lua --table-size=100000 --tables=10 --threads=100 --time=10 --mysql-db=testdb --mysql-host=${ARGS_SOURCE_HOST} --mysql-port=${ARGS_SOURCE_PORT} --mysql-user=${ARGS_SOURCE_USER} --mysql-password=${ARGS_SOURCE_PASSWORD} --db-driver=mysql $action
    done
done

# ./gh-ost \
#  --max-load=Threads_running=16 \
#  --critical-load=Threads_running=32 \
#  --chunk-size=1000 \
#  --initially-drop-old-table \
#  --initially-drop-ghost-table \
#  --initially-drop-socket-file \
#  --user="root" \
#  --password="root_password" \
#  --host='127.0.0.1' \
#  --port=33061 \
#  --database="test" \
#  --table="tab1" \
#  --verbose \
#  --allow-on-master \
#  --assume-master-host="127.0.0.1:33061" \
#  --aliyun-rds \
#  --alter="add column c varchar(100)" \
#  --assume-rbr \
#  --execute

# sleep 10

for i in `seq 1 10000`;
    mysql --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} -e 'show master status\G'
    mysql --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} -e 'show master status\G'
    sleep 1
done
# kill $MYSQLBINLOG_SYNC_PID
