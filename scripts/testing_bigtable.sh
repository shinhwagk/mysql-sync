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

echo "reset master"
MYSQL_SOURCE_CLIENT -e 'reset master;'
MYSQL_TARGET_CLIENT -e 'reset master;'

MYSQL_SOURCE_CLIENT -e "SHOW MASTER STATUS\G"
MYSQL_TARGET_CLIENT -e "SHOW MASTER STATUS\G"

MYSQL_SOURCE_CLIENT -e "CREATE DATABASE test;"
MYSQL_SOURCE_CLIENT -e "CREATE TABLE test.bigtable( id INT AUTO_INCREMENT PRIMARY KEY, c1 VARCHAR(2000), c2 VARCHAR(2000));"

col_val=$(tr -dc 'a-zA-Z0-9' < /dev/urandom | head -c 1000)

echo "generate data...."
for i in `seq 1 100`; do
    for j in `seq 1 100`; do
        MYSQL_SOURCE_CLIENT -e "INSERT INTO test.bigtable(c1,c2) VALUES('${col_val}','${col_val}')" >/dev/null 2>&1 &
    done
wait
echo "generate batch $i"
done

echo "sync source to target"
time python3.12 main1.py --mysql_source_connection_string="${ARGS_SOURCE_USER}/${ARGS_SOURCE_PASSWORD}@${ARGS_SOURCE_HOST}:${ARGS_SOURCE_PORT}" --mysql_source_server_id=9999 --mysql_source_report_slave="test" --mysql_source_slave_heartbeat=2 --mysql_target_connection_string="${ARGS_TARGET_USER}/${ARGS_TARGET_PASSWORD}@${ARGS_TARGET_HOST}:${ARGS_TARGET_PORT}"

mysqldump --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} --set-gtid-purged=OFF --compact >/tmp/source.sql
mysqldump --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} --set-gtid-purged=OFF --compact >/tmp/target.sql

diff /tmp/source.sql /tmp/target.sql; exit $?
