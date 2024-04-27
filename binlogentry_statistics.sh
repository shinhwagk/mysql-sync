#!/usr/bin/env bash

set -e

declare -A STATS_Table_Write_rows_counter
declare -A STATS_Table_Delete_rows_counter
declare -A STATS_Table_Update_rows_counter

declare -A STATS_Table_map_id



declare -i STATS_Write_rows=0
declare -i STATS_Delete_rows=0
declare -i STATS_Update_rows=0
declare -i STATS_Table_map=0
declare -i STATS_COMMIT=0
declare -i STATS_ROLLBACK=0
declare -i STATS_TIMESTAMP=0
declare -i STATS_at=0
declare -i STATS_BINLOG=0
declare -i STATS_lines=0

declare STATS_GTID_NEXT=""
declare STATS_BINLOGFILE=""

_last_seconds=$SECONDS

update_or_initialize_key() {
    local -n arr=$1
    local key=$2
    if [[ -z "${arr[$key]}" ]]; then
        arr[$key]=1
    else
        ((arr[$key]++))
    fi
}

function print_statistics() {
    echo "statistics ${SECONDS} {\"Write_rows\":${STATS_Write_rows},\"Delete_rows\":${STATS_Delete_rows},\"Update_rows\":${STATS_Update_rows},\"Table_map\":${STATS_Table_map},\"GTID_NEXT\":\"${STATS_GTID_NEXT}\",\"COMMIT\":${STATS_COMMIT},\"ROLLBACK\":${STATS_ROLLBACK},\"TIMESTAMP\":${STATS_TIMESTAMP},\"at\":${STATS_at},\"BINLOG\":${STATS_BINLOG},\"BINLOGFILE\":\"${STATS_BINLOGFILE}\",\"lines\":${STATS_lines}}" >&2
}

function print_statistics_interval() {
    local local_seconds=$SECONDS

    if ((local_seconds - _last_seconds >= 1)); then
        print_statistics
        _last_seconds=$local_seconds;
    fi
}

function statistics_binlogentry() {
    local binlogEntry="$1"

    if [[ "${binlogEntry:0:1}" == '#' ]]; then
        # binlogentry: # at 1534
        if [[ "${binlogEntry:0:5}" == "# at " ]]; then
            STATS_at=${binlogEntry:5}
        elif [[ "${binlogEntry:0:2}" == "#2" ]]; then
            read -ra parts <<< "$binlogEntry"

            case "${parts[9]}" in
                "Start:"|"Previous-GTIDs"|"Stop"|"GTID"|"Query"|"Xid")
                    :
                    ;;
                "Table_map:")
                    STATS_Table_map=$((STATS_Table_map+1))

                    # tab_name=$(echo "${parts[10]}" | tr -d '`')
                    # tab_id=$(echo "${parts[14]}")
                    # [[ -z "${STATS_Table_map_id[$tab_id]}" ]] && STATS_Table_map_id[$tab_id]="$tab_name"
                    ;;
                "Write_rows:")
                    tab_id=$(echo "${parts[12]}")
                    update_or_initialize_key "STATS_Table_Write_rows_counter" $tab_id
                    STATS_Write_rows=$((STATS_Write_rows+1))
                    ;;
                "Delete_rows:")
                    tab_id=$(echo "${parts[12]}")
                    update_or_initialize_key "STATS_Table_Delete_rows_counter" $tab_id
                    STATS_Delete_rows=$((STATS_Delete_rows+1))
                    ;;
                "Update_rows:")
                    tab_id=$(echo "${parts[12]}")
                    update_or_initialize_key "STATS_Table_Update_rows_counter" $tab_id
                    STATS_Update_rows=$((STATS_Update_rows+1))
                    ;;
                "Rotate")
                    if [[ "${parts[10]}" == "to" ]]; then
                        STATS_BINLOGFILE="${parts[11]}"
                    fi
                    ;;
                *)
                    echo "error: ${parts[9]} $binlogEntry" >&2
                    exit 1
                    ;;
            esac

        elif [[ "${binlogEntry:0:9}" == "#700101  " ]]; then
            read -ra parts <<< "$binlogEntry"

            if [[ "${parts[7]}" == "Rotate" && "${parts[8]}" == "to" ]]; then
                STATS_BINLOGFILE="${parts[9]}"
            elif [[ "${parts[9]}" == "Rotate" && "${parts[10]}" == "to" ]]; then
                STATS_BINLOGFILE="${parts[11]}"
            fi
        fi

    else
        # binlogentry: SET @@SESSION.GTID_NEXT= '6af9e1aa-0225-11ef-8fee-0242ac140002:14'/*!*/; 
        if [[ "${binlogEntry:0:26}" == "SET @@SESSION.GTID_NEXT= '" && "${binlogEntry: -7}" == "'/*!*/;" ]]; then
            gtid_next=$(echo "$binlogEntry" | grep -o "'.*'" | sed "s/'//g")
            STATS_GTID_NEXT="${gtid_next}"
        elif [[ "$binlogEntry" == 'COMMIT/*!*/;' ]]; then
            STATS_COMMIT=$((STATS_COMMIT+1))
        elif [[ "$binlogEntry" == 'ROLLBACK/*!*/;' ]]; then
            STATS_ROLLBACK=$((STATS_ROLLBACK+1))
        elif [[ "${binlogEntry:0:14}" == "SET TIMESTAMP=" && "${binlogEntry: -6}" == "/*!*/;" ]]; then
            STATS_TIMESTAMP="${binlogEntry:14:10}}"
        elif [[ "$binlogEntry" == "BINLOG '" ]]; then
            STATS_BINLOG=$((STATS_BINLOG+1))
        else
            :
        fi
        echo "$binlogEntry"
    fi

    print_statistics_interval
}


# while [[ "$#" -gt 0 ]]; do
#     case $1 in
#         --age) age="$2"; shift ;;
#         --name) name="$2"; shift ;;
#         *) echo "Unknown parameter passed: $1"; exit 1 ;;
#     esac
#     shift
# done


while IFS= read -r line; do
    # STATS_lines=$((STATS_lines+1))

    # if [[ $STATS_lines == 2 ]]; then
    #     echo "${binlogEntry}" | grep -Eo '8.[0-9].[0-9]+'
    # fi

    statistics_binlogentry "$line";
done

print_statistics


for key in "${!STATS_Table_Write_rows_counter[@]}"; do
    echo "$key: ${STATS_Table_Write_rows_counter[$key]}" >&2
done