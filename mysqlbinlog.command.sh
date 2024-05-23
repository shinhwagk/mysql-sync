mysqlbinlog --host=db1 --port=3306 --user=root --password=root_password --read-from-remote-source=BINLOG-DUMP-GTIDS --compression-algorithms=zstd --zstd-compression-level=3 --verify-binlog-checksum --to-last-log --connection-server-id=111 --verbose --verbose --idempotent --force-read --print-table-metadata mysql-bin.000001 >/dev/null

mysqlbinlog --host=db1 --port=3306 --user=root --password=root_password --read-from-remote-source=BINLOG-DUMP-GTIDS --compression-algorithms=zstd --zstd-compression-level=3 --verify-binlog-checksum --to-last-log --connection-server-id=11121 --verbose --verbose --idempotent --force-read --print-table-metadata mysql-bin.000001 >binlog/mysql-bin.8.0.36.sql 

mysqlbinlog --host=db1 --port=3306 --user=root --password=root_password --read-from-remote-source=BINLOG-DUMP-GTIDS --compression-algorithms=zstd --zstd-compression-level=3 --verify-binlog-checksum --to-last-log --connection-server-id=11121 --verbose --verbose --idempotent --force-read --print-table-metadata mysql-bin.000001 | cargo run >/dev/null

mysql -uroot -proot_password -hdb1 -e "reset master;"
mysqlbinlog --host=db1 --port=3306 --user=root --password=root_password --read-from-remote-source=BINLOG-DUMP-GTIDS --compression-algorithms=zstd --zstd-compression-level=3 --verify-binlog-checksum --stop-never --connection-server-id=111 --verbose --verbose --idempotent --force-read --print-table-metadata mysql-bin.000001

mysqlbinlog -S /data/mysql/litb_zeus_8_0/mysql.sock --user=repl --password=repl --read-from-remote-source=BINLOG-DUMP-GTIDS --compression-algorithms=zstd --zstd-compression-level=3 --verify-binlog-checksum --to-last-log --connection-server-id=11121 --verbose --verbose --idempotent --force-read --print-table-metadata mysql-bin.000029 >mysql-bin.log
