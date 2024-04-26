#!/usr/bin/env bash

# set -e

STATS_DATA='{"Write_rows":0,"Table_map":0,"GTID_NEXT":"","COMMIT":0,"ROLLBACK":0,"TIMESTAMP":0,"at":0,"BINLOG":0,"BINLOGFILE":""}'

function print_statistics() {
    echo "statistics $STATS_DATA" >&2
}

function update_statistics() {
    local key=$1
    local val=$2
    local typ=$3
    STATS_DATA=$(echo "$STATS_DATA" | jq -c ".${key} = (${val} | $typ)")

    print_statistics
}

function increment_statistics() {
    local key=$1
    STATS_DATA=$(echo "$STATS_DATA" | jq -c ".${key} += 1")

    print_statistics
}

function processBinlogEntry() {
    local binlogEntry="$1"

    if [[ "${binlogEntry:0:1}" == '#' ]]; then
        if [[ "${binlogEntry:0:5}" == "# at " ]]; then
            STATS_DATA=$(jq -c ".at = ${binlogEntry:5}" <<< "$STATS_DATA")
            print_statistics
        elif [[ "${binlogEntry:0:2}" == "#2" ]]; then

            read -ra parts <<< "$binlogEntry"

            if [ "${parts[9]}" == "Write_rows:" ]; then
                STATS_DATA=$(jq -c ".Write_rows += 1" <<< "$STATS_DATA")
                print_statistics
            elif [[ "${parts[9]}" == "Table_map:" ]]; then
                STATS_DATA=$(jq -c ".Table_map += 1" <<< "$STATS_DATA")
                print_statistics
            elif [[ "${parts[10]}" == "Rotate" && "${parts[11]}"  == "to" ]]; then
                STATS_DATA=$(jq -c --arg value "${parts[12]}" '.BINLOGFILE = $value' <<< "$STATS_DATA")
                print_statistics
            fi


        elif [[ "${binlogEntry:0:9}" == "#700101  " ]]; then
            read -ra parts <<< "$binlogEntry"
            if [[ "${parts[7]}" == "Rotate" && "${parts[8]}"  == "to" ]]; then
                STATS_DATA=$(jq -c --arg value "${parts[9]}" '.BINLOGFILE = $value' <<< "$STATS_DATA")
                print_statistics
            fi
        fi

    else
        if [[ "${binlogEntry:0:24}" == "SET @@SESSION.GTID_NEXT= '" && "${binlogEntry: -7}" == *"'/*!*/;" ]]; then
            :
        elif [[ "$binlogEntry" == "COMMIT/*!*/;" ]]; then
            STATS_DATA=$(jq -c ".COMMIT += 1" <<< "$STATS_DATA")
            print_statistics
        elif [[ "$binlogEntry" == "ROLLBACK/*!*/;" ]]; then
            STATS_DATA=$(jq -c ".ROLLBACK += 1" <<< "$STATS_DATA")
            print_statistics
        elif [[ "${binlogEntry:0:14}" == "SET TIMESTAMP=" && "${binlogEntry: -7}" == *"'/*!*/;" ]]; then
            STATS_DATA=$(jq -c ".TIMESTAMP = 11111" <<< "$STATS_DATA")
            print_statistics
        elif [[ "$binlogEntry" == "BINLOG '" ]]; then
            STATS_DATA=$(jq -c ".BINLOG += 1" <<< "$STATS_DATA")
            print_statistics
        else
            echo "$binlogEntry"
        fi
    fi
}

while IFS= read -r line; do
    processBinlogEntry "$line";
done

