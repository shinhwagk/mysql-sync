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

# ·
# echo "sync source to target"
# # time python3.12 main.py --config settings.py
# time go run src/*

rm -f /tmp/source.sql /tmp/target.sql
for db in $(MYSQL_SOURCE_CLIENT -e "SHOW DATABASES;" | grep -Ev "(Database|information_schema|performance_schema|mysql|sys)"); do
    mysqldump --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} --set-gtid-purged=OFF --compact --databases $db >>/tmp/source.sql
    mysqldump --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} --set-gtid-purged=OFF --compact --databases $db >>/tmp/target.sql
done

# mysqldump --host=${ARGS_SOURCE_HOST} --port=${ARGS_SOURCE_PORT} --user=${ARGS_SOURCE_USER} --password=${ARGS_SOURCE_PASSWORD} --set-gtid-purged=OFF --compact --ignore-database --all-databases  >/tmp/source.sql
# mysqldump --host=${ARGS_TARGET_HOST} --port=${ARGS_TARGET_PORT} --user=${ARGS_TARGET_USER} --password=${ARGS_TARGET_PASSWORD} --set-gtid-purged=OFF --compact --all-databases >/tmp/target.sql

MYSQL_SOURCE_CLIENT -e "SHOW MASTER STATUS\G"
MYSQL_TARGET_CLIENT -e "SHOW MASTER STATUS\G"

sha1sum /tmp/source.sql
sha1sum /tmp/target.sql


diff /tmp/source.sql /tmp/target.sql; exit $?
