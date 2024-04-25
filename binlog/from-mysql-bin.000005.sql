# The proper term is pseudo_replica_mode, but we use this compatibility alias
# to make the statement usable on server versions 8.0.24 and older.
/*!50530 SET @@SESSION.PSEUDO_SLAVE_MODE=1*/;
/*!50003 SET @OLD_COMPLETION_TYPE=@@COMPLETION_TYPE,COMPLETION_TYPE=0*/;
/*!50700 SET @@SESSION.RBR_EXEC_MODE=IDEMPOTENT*/;

DELIMITER /*!*/;
# at 4
#700101  0:00:00 server id 1  end_log_pos 0 	Rotate to mysql-bin.000004  pos: 4
# at 4
#240424 10:32:55 server id 1  end_log_pos 127 CRC32 0xfb73046c 	Start: binlog v 4, server v 8.3.0 created 240424 10:32:55
BINLOG '
198oZg8BAAAAewAAAH8AAAAAAAQAOC4zLjAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAEwANAAgAAAAABAAEAAAAYwAEGggAAAAICAgCAAAACgoKKioAEjQA
CigAAAFsBHP7
'/*!*/;
# at 127
#240424 10:32:55 server id 1  end_log_pos 198 CRC32 0xec0df8cd 	Previous-GTIDs
# 6af9e1aa-0225-11ef-8fee-0242ac140002:1-7
# at 198
#240424 10:33:55 server id 1  end_log_pos 275 CRC32 0xd15fee75 	GTID	last_committed=0	sequence_number=1	rbr_only=no	original_committed_timestamp=1713954835773264	immediate_commit_timestamp=1713954835773264	transaction_length=240
# original_commit_timestamp=1713954835773264 (2024-04-24 10:33:55.773264 UTC)
# immediate_commit_timestamp=1713954835773264 (2024-04-24 10:33:55.773264 UTC)
/*!80001 SET @@session.original_commit_timestamp=1713954835773264*//*!*/;
/*!80014 SET @@session.original_server_version=80300*//*!*/;
/*!80014 SET @@session.immediate_server_version=80300*//*!*/;
SET @@SESSION.GTID_NEXT= '6af9e1aa-0225-11ef-8fee-0242ac140002:8'/*!*/;
# at 275
#240424 10:33:55 server id 1  end_log_pos 438 CRC32 0xff9ecf22 	Query	thread_id=8	exec_time=0	error_code=0	Xid = 27
SET TIMESTAMP=1713954835/*!*/;
SET @@session.pseudo_thread_id=8/*!*/;
SET @@session.foreign_key_checks=1, @@session.sql_auto_is_null=0, @@session.unique_checks=1, @@session.autocommit=1/*!*/;
SET @@session.sql_mode=1168113696/*!*/;
SET @@session.auto_increment_increment=1, @@session.auto_increment_offset=1/*!*/;
/*!\C latin1 *//*!*/;
SET @@session.character_set_client=8,@@session.collation_connection=8,@@session.collation_server=255/*!*/;
SET @@session.lc_time_names=0/*!*/;
SET @@session.collation_database=DEFAULT/*!*/;
/*!80011 SET @@session.default_collation_for_utf8mb4=255*//*!*/;
GRANT REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO 'repl'@'%'
/*!*/;
# at 438
#240424 10:39:20 server id 1  end_log_pos 515 CRC32 0x9d4e0da3 	GTID	last_committed=1	sequence_number=2	rbr_only=no	original_committed_timestamp=1713955160734939	immediate_commit_timestamp=1713955160734939	transaction_length=185
# original_commit_timestamp=1713955160734939 (2024-04-24 10:39:20.734939 UTC)
# immediate_commit_timestamp=1713955160734939 (2024-04-24 10:39:20.734939 UTC)
/*!80001 SET @@session.original_commit_timestamp=1713955160734939*//*!*/;
/*!80014 SET @@session.original_server_version=80300*//*!*/;
/*!80014 SET @@session.immediate_server_version=80300*//*!*/;
SET @@SESSION.GTID_NEXT= '6af9e1aa-0225-11ef-8fee-0242ac140002:9'/*!*/;
# at 515
#240424 10:39:20 server id 1  end_log_pos 623 CRC32 0xf1b3a5c6 	Query	thread_id=8	exec_time=0	error_code=0	Xid = 37
SET TIMESTAMP=1713955160/*!*/;
/*!80016 SET @@session.default_table_encryption=0*//*!*/;
create database test
/*!*/;
# at 623
#240424 10:39:30 server id 1  end_log_pos 700 CRC32 0xed330342 	GTID	last_committed=2	sequence_number=3	rbr_only=no	original_committed_timestamp=1713955170854489	immediate_commit_timestamp=1713955170854489	transaction_length=240
# original_commit_timestamp=1713955170854489 (2024-04-24 10:39:30.854489 UTC)
# immediate_commit_timestamp=1713955170854489 (2024-04-24 10:39:30.854489 UTC)
/*!80001 SET @@session.original_commit_timestamp=1713955170854489*//*!*/;
/*!80014 SET @@session.original_server_version=80300*//*!*/;
/*!80014 SET @@session.immediate_server_version=80300*//*!*/;
SET @@SESSION.GTID_NEXT= '6af9e1aa-0225-11ef-8fee-0242ac140002:10'/*!*/;
# at 700
#240424 10:39:30 server id 1  end_log_pos 863 CRC32 0x248ba1b1 	Query	thread_id=8	exec_time=0	error_code=0	Xid = 38
SET TIMESTAMP=1713955170/*!*/;
GRANT REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO 'repl'@'%'
/*!*/;
# at 863
#240424 10:39:35 server id 1  end_log_pos 940 CRC32 0xe713bb39 	GTID	last_committed=3	sequence_number=4	rbr_only=no	original_committed_timestamp=1713955175910441	immediate_commit_timestamp=1713955175910441	transaction_length=240
# original_commit_timestamp=1713955175910441 (2024-04-24 10:39:35.910441 UTC)
# immediate_commit_timestamp=1713955175910441 (2024-04-24 10:39:35.910441 UTC)
/*!80001 SET @@session.original_commit_timestamp=1713955175910441*//*!*/;
/*!80014 SET @@session.original_server_version=80300*//*!*/;
/*!80014 SET @@session.immediate_server_version=80300*//*!*/;
SET @@SESSION.GTID_NEXT= '6af9e1aa-0225-11ef-8fee-0242ac140002:11'/*!*/;
# at 940
#240424 10:39:35 server id 1  end_log_pos 1103 CRC32 0x90f4f250 	Query	thread_id=8	exec_time=0	error_code=0	Xid = 39
SET TIMESTAMP=1713955175/*!*/;
GRANT REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO 'repl'@'%'
/*!*/;
# at 1103
#240424 10:40:07 server id 1  end_log_pos 1180 CRC32 0xd5ea46bf 	GTID	last_committed=4	sequence_number=5	rbr_only=no	original_committed_timestamp=1713955207175019	immediate_commit_timestamp=1713955207175019	transaction_length=191
# original_commit_timestamp=1713955207175019 (2024-04-24 10:40:07.175019 UTC)
# immediate_commit_timestamp=1713955207175019 (2024-04-24 10:40:07.175019 UTC)
/*!80001 SET @@session.original_commit_timestamp=1713955207175019*//*!*/;
/*!80014 SET @@session.original_server_version=80300*//*!*/;
/*!80014 SET @@session.immediate_server_version=80300*//*!*/;
SET @@SESSION.GTID_NEXT= '6af9e1aa-0225-11ef-8fee-0242ac140002:12'/*!*/;
# at 1180
#240424 10:40:07 server id 1  end_log_pos 1294 CRC32 0x74c30455 	Query	thread_id=8	exec_time=0	error_code=0	Xid = 42
SET TIMESTAMP=1713955207/*!*/;
/*!80013 SET @@session.sql_require_primary_key=0*//*!*/;
create table test.tab1 (a int)
/*!*/;
# at 1294
#240424 10:43:18 server id 1  end_log_pos 1373 CRC32 0xb7be6414 	GTID	last_committed=5	sequence_number=6	rbr_only=yes	original_committed_timestamp=1713955398176081	immediate_commit_timestamp=1713955398176081	transaction_length=271
/*!50718 SET TRANSACTION ISOLATION LEVEL READ COMMITTED*//*!*/;
# original_commit_timestamp=1713955398176081 (2024-04-24 10:43:18.176081 UTC)
# immediate_commit_timestamp=1713955398176081 (2024-04-24 10:43:18.176081 UTC)
/*!80001 SET @@session.original_commit_timestamp=1713955398176081*//*!*/;
/*!80014 SET @@session.original_server_version=80300*//*!*/;
/*!80014 SET @@session.immediate_server_version=80300*//*!*/;
SET @@SESSION.GTID_NEXT= '6af9e1aa-0225-11ef-8fee-0242ac140002:13'/*!*/;
# at 1373
#240424 10:43:18 server id 1  end_log_pos 1444 CRC32 0xd2c4a75e 	Query	thread_id=8	exec_time=0	error_code=0
SET TIMESTAMP=1713955398/*!*/;
BEGIN
/*!*/;
# at 1444
#240424 10:43:18 server id 1  end_log_pos 1494 CRC32 0xd4fac88c 	Table_map: `test`.`tab1` mapped to number 92
# has_generated_invisible_primary_key=0
# at 1494
#240424 10:43:18 server id 1  end_log_pos 1534 CRC32 0xa03d6570 	Write_rows: table id 92 flags: STMT_END_F

BINLOG '
RuIoZhMBAAAAMgAAANYFAAAAAFwAAAAAAAEABHRlc3QABHRhYjEAAQMAAQEBAIzI+tQ=
RuIoZh4BAAAAKAAAAP4FAAAAAFwAAAAAAAEAAgAB/wABAAAAcGU9oA==
'/*!*/;
### INSERT INTO `test`.`tab1`
### SET
###   @1=1
# at 1534
#240424 10:43:18 server id 1  end_log_pos 1565 CRC32 0x214a85fe 	Xid = 45
COMMIT/*!*/;
# at 1565
#240424 10:45:54 server id 1  end_log_pos 1644 CRC32 0x053e2440 	GTID	last_committed=6	sequence_number=7	rbr_only=yes	original_committed_timestamp=1713955554358471	immediate_commit_timestamp=1713955554358471	transaction_length=361
/*!50718 SET TRANSACTION ISOLATION LEVEL READ COMMITTED*//*!*/;
# original_commit_timestamp=1713955554358471 (2024-04-24 10:45:54.358471 UTC)
# immediate_commit_timestamp=1713955554358471 (2024-04-24 10:45:54.358471 UTC)
/*!80001 SET @@session.original_commit_timestamp=1713955554358471*//*!*/;
/*!80014 SET @@session.original_server_version=80300*//*!*/;
/*!80014 SET @@session.immediate_server_version=80300*//*!*/;
SET @@SESSION.GTID_NEXT= '6af9e1aa-0225-11ef-8fee-0242ac140002:14'/*!*/;
# at 1644
#240424 10:45:51 server id 1  end_log_pos 1715 CRC32 0x03885e83 	Query	thread_id=8	exec_time=0	error_code=0
SET TIMESTAMP=1713955551/*!*/;
BEGIN
/*!*/;
# at 1715
#240424 10:45:51 server id 1  end_log_pos 1765 CRC32 0x002f8c2a 	Table_map: `test`.`tab1` mapped to number 92
# has_generated_invisible_primary_key=0
# at 1765
#240424 10:45:51 server id 1  end_log_pos 1805 CRC32 0x84aa97a4 	Write_rows: table id 92 flags: STMT_END_F

BINLOG '
3+IoZhMBAAAAMgAAAOUGAAAAAFwAAAAAAAEABHRlc3QABHRhYjEAAQMAAQEBACqMLwA=
3+IoZh4BAAAAKAAAAA0HAAAAAFwAAAAAAAEAAgAB/wACAAAApJeqhA==
'/*!*/;
### INSERT INTO `test`.`tab1`
### SET
###   @1=2
# at 1805
#240424 10:45:52 server id 1  end_log_pos 1855 CRC32 0x0429d82d 	Table_map: `test`.`tab1` mapped to number 92
# has_generated_invisible_primary_key=0
# at 1855
#240424 10:45:52 server id 1  end_log_pos 1895 CRC32 0x02b44744 	Write_rows: table id 92 flags: STMT_END_F

BINLOG '
4OIoZhMBAAAAMgAAAD8HAAAAAFwAAAAAAAEABHRlc3QABHRhYjEAAQMAAQEBAC3YKQQ=
4OIoZh4BAAAAKAAAAGcHAAAAAFwAAAAAAAEAAgAB/wADAAAAREe0Ag==
'/*!*/;
### INSERT INTO `test`.`tab1`
### SET
###   @1=3
# at 1895
#240424 10:45:54 server id 1  end_log_pos 1926 CRC32 0xf205fd66 	Xid = 48
COMMIT/*!*/;
# at 1926
#240424 10:58:23 server id 1  end_log_pos 1973 CRC32 0x9022c10b 	Rotate to mysql-bin.000005  pos: 4
# at 4
#700101  0:00:00 server id 1  end_log_pos 0 CRC32 0x6fc1ca06 	Rotate to mysql-bin.000005  pos: 4
SET @@SESSION.GTID_NEXT= 'AUTOMATIC' /* added by mysqlbinlog */ /*!*/;
# at 4
#240424 10:58:23 server id 1  end_log_pos 127 CRC32 0x42d10f27 	Start: binlog v 4, server v 8.3.0 created 240424 10:58:23
BINLOG '
z+UoZg8BAAAAewAAAH8AAAAAAAQAOC4zLjAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAEwANAAgAAAAABAAEAAAAYwAEGggAAAAICAgCAAAACgoKKioAEjQA
CigAAAEnD9FC
'/*!*/;
# at 127
#240424 10:58:23 server id 1  end_log_pos 198 CRC32 0x910adac0 	Previous-GTIDs
# 6af9e1aa-0225-11ef-8fee-0242ac140002:1-14
# at 4
#700101  0:00:00 server id 1  end_log_pos 0 CRC32 0xf6c89bbc 	Rotate to mysql-bin.000006  pos: 4
SET @@SESSION.GTID_NEXT= 'AUTOMATIC' /* added by mysqlbinlog */ /*!*/;
# at 4
#240425  2:35:56 server id 1  end_log_pos 127 CRC32 0x1b608bd9 	Start: binlog v 4, server v 8.3.0 created 240425  2:35:56 at startup
ROLLBACK/*!*/;
BINLOG '
jMEpZg8BAAAAewAAAH8AAAAAAAQAOC4zLjAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAACMwSlmEwANAAgAAAAABAAEAAAAYwAEGggAAAAICAgCAAAACgoKKioAEjQA
CigAAAHZi2Ab
'/*!*/;
# at 127
#240425  2:35:56 server id 1  end_log_pos 198 CRC32 0xa3e331a9 	Previous-GTIDs
# 6af9e1aa-0225-11ef-8fee-0242ac140002:1-14
