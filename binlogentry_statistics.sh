#!/usr/bin/env bash

set -e

STATS_Write_rows=0
STATS_Table_map=0
STATS_GTID_NEXT=""
STATS_COMMIT=0
STATS_ROLLBACK=0
STATS_TIMESTAMP=0
STATS_at=0
STATS_BINLOG=0
STATS_BINLOGFILE=""
STATS_lines=0

_last_unixtime=$SECONDS

function print_statistics() {
    local current_unixtime=$SECONDS

    if ((current_unixtime - _last_unixtime >= 1)); then
        echo "statistics ${current_unixtime} {\"Write_rows\":${STATS_Write_rows},\"Table_map\":${STATS_Table_map},\"GTID_NEXT\":\"${STATS_GTID_NEXT}\",\"COMMIT\":${STATS_COMMIT},\"ROLLBACK\":${STATS_ROLLBACK},\"TIMESTAMP\":${STATS_TIMESTAMP},\"at\":${STATS_at},\"BINLOG\":${STATS_BINLOG},\"BINLOGFILE\":\"${STATS_BINLOGFILE}\",\"lines\":${STATS_lines}}" >&2

        _last_unixtime=$current_unixtime;
    fi
}


function processBinlogEntry() {
    local binlogEntry="$1"

    STATS_lines=$((STATS_lines+1))

    if [[ "${binlogEntry:0:1}" == '#' ]]; then
        if [[ "${binlogEntry:0:5}" == "# at " ]]; then
            STATS_at=${binlogEntry:5}
            echo $STATS_lines $binlogEntry  ${binlogEntry:5} >&2
        elif [[ "${binlogEntry:0:2}" == "#2" ]]; then
            read -ra parts <<< "$binlogEntry"

            case "${parts[9]}" in
                "Start:"|"Previous-GTIDs"|"Stop"|"GTID"|"Query"|"Xid")
                    :
                    ;;
                "Table_map:")
                    STATS_Table_map=$((STATS_Table_map+1))
                    ;;
                "Write_rows:")
                    STATS_Write_rows=$((STATS_Write_rows+1))
                    ;;
                "Rotate")
                    if [[ "${parts[10]}" == "to" ]]; then
                        STATS_BINLOGFILE="${parts[11]}"
                    fi
                    ;;
                *)
                    echo "error: ${parts[9]} $binlogEntry"
                    exit 1
                    ;;
            esac

        elif [[ "${binlogEntry:0:9}" == "#700101  " ]]; then
            read -ra parts <<< "$binlogEntry"

            if [[ "${parts[7]}" == "Rotate" && "${parts[8]}" == "to" ]]; then
                STATS_BINLOGFILE="${parts[9]}"
            fi
        fi

    else
        if [[ "${binlogEntry:0:26}" == "SET @@SESSION.GTID_NEXT= '" && "${binlogEntry: -7}" == "'/*!*/;" ]]; then
            gtid_next=$(echo "$binlogEntry" | grep -o "'.*'" | sed "s/'//g")
            STATS_GTID_NEXT="${gtid_next}"
        elif [[ "$binlogEntry" == "COMMIT/*!*/;" ]]; then
            STATS_COMMIT=$((STATS_COMMIT+1))
        elif [[ "$binlogEntry" == "ROLLBACK/*!*/;" ]]; then
            STATS_ROLLBACK=$((STATS_ROLLBACK+1))
        elif [[ "${binlogEntry:0:14}" == "SET TIMESTAMP=" && "${binlogEntry: -6}" == "/*!*/;" ]]; then
            STATS_TIMESTAMP="${binlogEntry#*TIMESTAMP=}"
            STATS_TIMESTAMP="${STATS_TIMESTAMP%%.*}"
            STATS_TIMESTAMP="${STATS_TIMESTAMP%%/*}"
        elif [[ "$binlogEntry" == "BINLOG '" ]]; then
            STATS_BINLOG=$((STATS_BINLOG+1))
        else
            :
        fi
        echo "$binlogEntry"
    fi

    print_statistics
}


while IFS= read -r line; do
    processBinlogEntry "$line";
done

print_statistics