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

cat scripts/mysql.sql | MYSQL_SOURCE_CLIENT

echo "start load data to source database"
function sysbench_load_data() {
    local testdb=$1
    MYSQL_SOURCE_CLIENT -e "CREATE DATABASE IF NOT EXISTS ${testdb};"
    # for testname in oltp_insert oltp_delete oltp_update_index oltp_update_non_index oltp_write_only bulk_insert; do
    for testname in oltp_insert; do
        for action in cleanup prepare; do
            echo "sysbench ${testdb}-${testname}-${action} start."
            sysbench /usr/share/sysbench/${testname}.lua --table-size=1000 --tables=10 --threads=10 --time=10 --mysql-db=${testdb} --mysql-host=${ARGS_SOURCE_HOST} --mysql-port=${ARGS_SOURCE_PORT} --mysql-user=${ARGS_SOURCE_USER} --mysql-password=${ARGS_SOURCE_PASSWORD} --db-driver=mysql $action >/dev/null
            echo "sysbench ${testdb}-${testname}-${action} done."
        done
    done
}

for dbid in `seq 1 1`; do
    time sysbench_load_data "testdb_${dbid}"
done


# echo "sync source to target"
# time python3.12 main.py --config settings.py

# rm -f /tmp/source.sql /tmp/target.sql
# for db in $(MYSQL_SOURCE_CLIENT -e "SHOW DATABASES;" | grep -Ev "(Database|information_schema|performance_schema|mysql|sys)"); do
#     mysqldump --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} --set-gtid-purged=OFF --compact --databases $db >>/tmp/source.sql
#     mysqldump --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} --set-gtid-purged=OFF --compact --databases $db >>/tmp/target.sql
# done

# # mysqldump --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} --set-gtid-purged=OFF --compact --ignore-database --all-databases  >/tmp/source.sql
# # mysqldump --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} --set-gtid-purged=OFF --compact --all-databases >/tmp/target.sql

# MYSQL_SOURCE_CLIENT -e "SHOW MASTER STATUS\G"
# MYSQL_TARGET_CLIENT -e "SHOW MASTER STATUS\G"

# sha1sum /tmp/source.sql
# sha1sum /tmp/target.sql


# diff /tmp/source.sql /tmp/target.sql; exit $?
