```sql
create user repl@'%' IDENTIFIED BY 'example';
grant all on *.* to repl@'%';

CHANGE MASTER TO
  MASTER_HOST='db1',
  MASTER_PORT=3306,
  MASTER_USER='repl',
  MASTER_PASSWORD='example',
  MASTER_AUTO_POSITION = 1;

create database test;
create table test.tab1(a int primary key, b varchar(10));
insert into test.tab1 values(1,'a');
```

```sh
mysqlbinlog -hdb --user=repl --password=repl --read-from-remote-server --stop-never -v mysql-bin.000001 | mysql -hdb2 -urepl --password=example

mysqlbinlog -hdb --user=repl --password=repl --read-from-remote-server --stop-never -v -i mysql-bin.000001 | mysql -v -hdb2 -urepl --password=example

mysqlbinlog -hdb --user=repl --password=repl --read-from-remote-server --stop-never -v -i --bind-address mysql-bin.000009 

mysqlbinlog -hdb --user=repl --password=repl --read-from-remote-server --stop-never -v -i --stop-never-slave-server-id=1111 mysql-bin.000009 

mysqlbinlog -hdb --user=repl --password=repl --read-from-remote-server --stop-never -v -i --exclude-gtids=6af9e1aa-0225-11ef-8fee-0242ac140002:1-12 mysql-bin.000001



mysqlbinlog -hdb --user=repl --password=repl --read-from-remote-server --stop-never -v --start-position=12345 mysql-bin.000006




mysqlbinlog -hdb --user=root --password=example --read-from-remote-server --stop-never -vv mysql-bin.000005

mysqlbinlog -hdb --user=repl --password=example --read-from-remote-server --stop-never -vv --base64-output=DECODE-ROWS mysql-bin.000008 | mysql -v -hdb2 -urepl --password=example

mysqlbinlog -hdb --user=root --password=example --read-from-remote-server --stop-never -vv --base64-output=DECODE-ROWS mysql-bin.000008 | mysql -v -hdb2 -urepl --password=example


mysqlbinlog -vv --base64-output=DECODE-ROWS --host=127.0.0.1 --port=3306 --user=root --password=rootpassword mysql-bin.000001 | mysql -u target_user -p'target_password' -h target_host target_database


mysqlbinlog --host=db1 --port=3306 --user=root --password=example --read-from-remote-server --to-last-log --verbose --verbose --idempotent --force-read --print-table-metadata mysql-bin.000001 >binlog/from-mysql-bin.000001.all-parameter.sql

mysqlbinlog --host=db1 --port=3306 --user=root --password=example --read-from-remote-source=BINLOG-DUMP-NON-GTIDS --to-last-log --verbose --verbose --idempotent --force-read --print-table-metadata mysql-bin.000001 >binlog/from-mysql-bin.000001.BINLOG-DUMP-NON-GTIDS.sql
mysqlbinlog --host=db1 --port=3306 --user=root --password=example --read-from-remote-source=BINLOG-DUMP-GTIDS --compression-algorithms=zstd --zstd-compression-level=3 --to-last-log --verbose --verbose --idempotent --force-read --print-table-metadata mysql-bin.000001 >binlog/from-mysql-bin.000001.BINLOG-DUMP-GTIDS.sql

# --verify-binlog-checksum
mysqlbinlog --host=db1 --port=3306 --user=root --password=example --read-from-remote-source=BINLOG-DUMP-GTIDS --compression-algorithms=zstd --zstd-compression-level=3 --verify-binlog-checksum --to-last-log --verbose --verbose --idempotent --force-read --print-table-metadata mysql-bin.000001 >binlog/from-mysql-bin.000001.verify-binlog-checksum.sql

# --stop-never-slave-server-id
mysqlbinlog --host=db1 --port=3306 --user=root --password=example --read-from-remote-source=BINLOG-DUMP-GTIDS --compression-algorithms=zstd --zstd-compression-level=3 --verify-binlog-checksum --to-last-log --connection-server-id=1111 --stop-never --verbose --verbose --idempotent --force-read --print-table-metadata mysql-bin.000001 >/dev/null

# --connection-server-id
mysqlbinlog --host=db1 --port=3306 --user=root --password=example --read-from-remote-source=BINLOG-DUMP-GTIDS --compression-algorithms=zstd --zstd-compression-level=3 --verify-binlog-checksum --to-last-log --connection-server-id=111--verbose --verbose --idempotent --force-read --print-table-metadata mysql-bin.000001 >/dev/null

mysqlbinlog --host=db1 --port=3306 --user=root --password=example --read-from-remote-source=BINLOG-DUMP-GTIDS --compression-algorithms=zstd --zstd-compression-level=3 --verify-binlog-checksum --to-last-log --connection-server-id=11121 --verbose --verbose --idempotent --force-read --print-table-metadata mysql-bin.000001 >binlog/mysql-bin.8.0.36.sql 

mysqlbinlog --host=db1 --port=3306 --user=root --password=example --read-from-remote-source=BINLOG-DUMP-GTIDS --compression-algorithms=zstd --zstd-compression-level=3 --verify-binlog-checksum --to-last-log --connection-server-id=11121 --verbose --verbose --idempotent --force-read --print-table-metadata mysql-bin.000001 | cargo run >/dev/null

#
/usr/local/Percona-Server-8.0.34-26/bin/mysqlbinlog -S /data/mysql/litb_zeus_8_0/mysql.sock --user=repl --password=repl --read-from-remote-source=BINLOG-DUMP-GTIDS --compression-algorithms=zstd --zstd-compression-level=3 --verify-binlog-checksum --to-last-log --connection-server-id=11121 --verbose --verbose --idempotent --force-read --print-table-metadata mysql-bin.000029 >mysql-bin.log



```

```sh
docker run -d -e MYSQL_ROOT_PASSWORD=example -e MYSQL_server_id=999  mysql:8.0.36 

echo 'hello world' | deno run --allow-read --allow-write transform.ts | wc -c

# # source
# nc -l 12345 | while true; do
#     gunzip | app_process
# done

# while true; do
# sleep 1
# echo aaa | gzip | nc 127.0.0.1 12345
# done 

# # target
# nc -l 12345 | gunzip | wc -l


generate_data() {
    count=0
    while true; do
        echo  "DataDataDataDataDDataDataDataDataDDataDataDataDataDDataDataDataDataDDataDataDataDataDDataDataDataDataDDataDataDataDataDDataDataDataDataDDataDataDataDataDDataDataDataDataDDataDataDataDataDDataDataDataDataD  $count"
        count=$((count+1))
        sleep 1
    done
}

generate_data | nc 127.0.0.1 12345
nc -k -l 12345
```

### test

- mysql version: 8.0.24
- deno version: v1.42.4

```sh
gcc main.c -o main
generate_data() {
  mysqlbinlog -hdb --user=repl --password=repl --read-from-remote-server --stop-never -v -i mysql-bin.000001
} | ./main
```

## links
- https://dev.mysql.com/doc/refman/8.0/en/mysqlbinlog-hexdump.html
- https://dev.mysql.com/doc/refman/8.0/en/mysqlbinlog.html