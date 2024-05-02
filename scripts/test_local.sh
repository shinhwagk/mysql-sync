#!/bin/bash

mkdir -p temp

rm -fr temp/*

kill -9 $(ps -ef | grep python | grep main.py -w  | grep -v grep | awk '{print $2}')

/usr/bin/python3.11 /workspaces/mysql-mysqlbinlog-replicaiton/generate_sql_statements.py | mysql -uroot -pexample -hdb1
/usr/bin/python3.11 /workspaces/mysql-mysqlbinlog-replicaiton/generate_sql_statements.py | mysql -uroot -pexample -hdb2
mysql -uroot -pexample -hdb1 -e "reset master;";
mysql -uroot -pexample -hdb2 -e "reset master;";
cargo build

cd temp
/usr/bin/python3.11 /workspaces/mysql-mysqlbinlog-replicaiton/main.py --source-dsn root/example@db1:3306 --target-dsn root/example@db2:3306 &

sleep 5

for i in `seq 1 10` do
  /usr/bin/python3.11 /workspaces/mysql-mysqlbinlog-replicaiton/generate_sql_statements.py | mysql -uroot -pexample -hdb1
  mysql -uroot -pexample -hdb1 -e "show master status\G";
  mysql -uroot -pexample -hdb2 -e "show master status\G";
  sleep 2;
done
