#!/bin/bash

STATS_DATA='{"Write_rows":0,"Table_map":0,"GTID_NEXT":"","COMMIT":0,"ROLLBACK":0,"TIMESTAMP":0,"at":0,"BINLOG":0,"BINLOGFILE":"",lines:0}'

# function processBinlogEntry(){
#     local binlogEntry="$1"
#      if [[ "${binlogEntry}" == 'abc' ]]; then
#         abc=$(jq -c ".abc +=1" <<< "$abc")
#         echo $abc
#     fi
#     echo $abc
# }

function processBinlogEntry() {
    local binlogEntry="$1"

    if [[ "${binlogEntry:0:1}" == '#' ]]; then

        STATS_DATA=$(jq -c ".Write_rows +=1" <<< "$STATS_DATA")
        echo $STATS_DATA

    else
        echo $STATS_DATA >&2
    fi
}


while IFS= read -r line; do
    processBinlogEntry "$line"
done