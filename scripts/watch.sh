#!/bin/bash
while true; do
    mysql -uroot -pexample -hdb1  -e "show master status \G" | grep Executed_Gtid_Set
    mysql -uroot -pexample -hdb2  -e "show master status \G" | grep Executed_Gtid_Set
    sleep 1
done