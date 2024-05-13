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
function test() {
  BINLOGFILE=$1
  mysqlbinlog --host=db1 --port=3306 --user=root --password=example --read-from-remote-source=BINLOG-DUMP-GTIDS --compression-algorithms=zstd --zstd-compression-level=3 --verify-binlog-checksum --to-last-log --connection-server-id=11121 --verbose --verbose --idempotent --force-read --print-table-metadata $BINLOGFILE
}


#

python3.11 main.py --source-dsn root/example@db1:3306 --target-dsn root/example@db2:3306

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

### suppert data types
- VARCHAR
- CHAR
- INT
- SMALLINT
- TINYINT
- BIGINT
- FLOAT
- DOUBLE
- DECIMAL
- DATE
- TIME
- DATETIME
- TIMESTAMP
- YEAR
- BLOB
- TEXT
- ENUM
- SET

## links
- https://dev.mysql.com/doc/refman/8.0/en/mysqlbinlog-hexdump.html
- https://dev.mysql.com/doc/refman/8.0/en/mysqlbinlog.html
- https://www.baeldung.com/linux/pipe-buffer-capacity
- https://www.mydbops.com/blog/fastest-parallel-replication-method-in-mysql-8/