#!/usr/bin/env bash

set -eo pipefail

declare source_dsn=""
declare target_dsn=""
declare mysqlbinlog_statistics=""
declare mysqlbinlog_zstd_compression_level=""
declare mysqlbinlog_connection_server_id="99999"
declare mysqlbinlog_exclude_gtids=""
declare -i mysqlbinlog_stop_never=0

if [[ $# -lt 4 ]]; then
    echo "Usage: $0 --source-dsn <source_dsn> --target-dsn <target_dsn> [OPTIONS]"
    exit 1
fi

while [ "$#" -gt 0 ]; do
    case "$1" in
        --source-dsn)
            source_dsn="$2"
            shift 2
            ;;
        --target-dsn)
            target_dsn="$2"
            shift 2
            ;;
        --mysqlbinlog-statistics)
            mysqlbinlog_statistics="$2"
            shift 2
            ;;
        --mysqlbinlog-zstd-compression-level)
            mysqlbinlog_zstd_compression_level="$2"
            shift 2
            ;;
        --mysqlbinlog-connection-server-id)
            if [[ -n "$2" ]]; then
              mysqlbinlog_connection_server_id="$2"
            fi
            shift 2
            ;;
        --mysqlbinlog-exclude-gtids)
            mysqlbinlog_exclude_gtids="$2"
            shift 2
            ;;
        --mysqlbinlog-stop-never)
            mysqlbinlog_stop_never=1
            shift
            ;;
        *)
            echo "Invalid option: $1"
            exit 1
            ;;
    esac
done

cleanup() {
    echo "Received termination signal. Cleaning up..."
    for pid in `pgrep -P $$`; do
        [[ "`ps -o cmd= -p $pid | awk '{print $1}'`" == "mysqlbinlog" ]] && kill $pid
    done
}

trap cleanup SIGINT SIGTERM

function parse_connection_string() {
    local dsn=$1
    local user="${dsn%%/*}"
    local pass_and_host="${dsn#*/}"
    local password="${pass_and_host%%@*}"
    local host_and_port="${pass_and_host#*@}"
    local host="${host_and_port%:*}"
    local port="${host_and_port##*:}"
    local output="--host=$host --port=$port --user=$user --password=$password"
    echo "$output"
}

function generate_delay_data() {
  local connection_string="$(parse_connection_string $source_dsn)";
  mysql $connection_string -N -s -e "CREATE DATABASE IF NOT EXISTS mysqlbinlogsyncl;"
  mysql $connection_string -N -s -e "CREATE TABLE IF NOT EXISTS mysqlbinlogsync.sync_table (id INT PRIMARY KEY, ts DATETIME);"
  
  while true; do
    mysql $connection_string -N -s -e "REPLACE INTO mysqlbinlogsync.sync_table VALUES(1, now());" &>/dev/null
    sleep 0.5
  done
}

function query_executed_gtid_set() {
  local connection_string="$(parse_connection_string $target_dsn)";
#   mysql $connection_string -N -s -e "show master status\G" 2>/dev/null | grep Executed_Gtid_Set | awk '{print $2}'
  mysql $connection_string -N -s -e "show master status\G" 2>/dev/null | tail -n 1
}   

function query_first_master_log_file() {
  local connection_string="$(parse_connection_string $source_dsn)";
  mysql $connection_string -N -s -e 'show master logs' 2>/dev/null | awk '{print $1}'
}

function capture_timestamp() {
    local output_file="temp/xxxx"
    local regex1='^SET TIMESTAMP=([0-9]+)'
    local regex2='^#2'
    
    while IFS= read -r line; do
        if [[ "$line" =~ $regex1 ]]; then
            echo "${BASH_REMATCH[1]}" > "$output_file"
            :
        elif [[ "$line" =~ $regex2 ]]; then
            echo $line
            date -d "${line:1:16}" +%s > "$output_file"
        fi
    done
}

function main_sync() {
  local options=" --read-from-remote-source=BINLOG-DUMP-GTIDS --verify-binlog-checksum --connection-server-id=${mysqlbinlog_connection_server_id} --verbose --verbose --idempotent --force-read --print-table-metadata"
  if [[ -n "$mysqlbinlog_zstd_compression_level" ]]; then
    options+=" --compression-algorithms=zstd --zstd-compression-level=${mysqlbinlog_zstd_compression_level}"
  fi

  if [[ -n "$mysqlbinlog_exclude_gtids" ]]; then
      options+=" --exclude-gtids=\"${mysqlbinlog_exclude_gtids}\""
  else
    local target_executed_gtid_set=$(query_executed_gtid_set)

    echo $target_executed_gtid_set

    if [[ ! -z $target_executed_gtid_set ]]; then
        options+=" --exclude-gtids=${target_executed_gtid_set}"
    fi
  fi

  if [[ "$mysqlbinlog_stop_never" = 1 ]]; then
      options+=" --stop-never"
  else
      options+=" --to-last-log"
  fi

  options+=" ${start_binlogfile}"

  echo "mysqlbinlog $(parse_connection_string $source_dsn) $options $(query_first_master_log_file)"
  echo "mysql $(parse_connection_string $target_dsn) --verbose --verbose --verbose"

  mysqlbinlog $(parse_connection_string $source_dsn) $options $(query_first_master_log_file) | mysql $(parse_connection_string $target_dsn) --verbose --verbose --verbose | capture_timestamp
}

main_sync
