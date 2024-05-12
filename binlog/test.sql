# The proper term is pseudo_replica_mode, but we use this compatibility alias
# to make the statement usable on server versions 8.0.24 and older.
/*!50530 SET @@SESSION.PSEUDO_SLAVE_MODE=1*/;
/*!50003 SET @OLD_COMPLETION_TYPE=@@COMPLETION_TYPE,COMPLETION_TYPE=0*/;
/*!50700 SET @@SESSION.RBR_EXEC_MODE=IDEMPOTENT*/;

DELIMITER /*!*/;
# at 4
#700101  0:00:00 server id 1  end_log_pos 0 	Rotate to mysql-bin.000001  pos: 4
# at 4
#240505  7:12:23 server id 1  end_log_pos 126 CRC32 0xfbfc0917 	Start: binlog v 4, server v 8.0.36 created 240505  7:12:23 at startup
ROLLBACK/*!*/;
# at 126
#240505  7:12:23 server id 1  end_log_pos 157 CRC32 0xb776996f 	Previous-GTIDs
# [empty]
# at 157
#240505  7:26:40 server id 1  end_log_pos 234 CRC32 0x18c9d5e5 	GTID	last_committed=0	sequence_number=1	rbr_only=no	original_committed_timestamp=1714894000491564	immediate_commit_timestamp=1714894000491564	transaction_length=185
# original_commit_timestamp=1714894000491564 (2024-05-05 07:26:40.491564 UTC)
# immediate_commit_timestamp=1714894000491564 (2024-05-05 07:26:40.491564 UTC)
/*!80001 SET @@session.original_commit_timestamp=1714894000491564*//*!*/;
/*!80014 SET @@session.original_server_version=80036*//*!*/;
/*!80014 SET @@session.immediate_server_version=80036*//*!*/;
SET @@SESSION.GTID_NEXT= '7f4be0b0-09b9-11ef-9bc6-0242ac130002:1'/*!*/;
# at 234
#240505  7:26:40 server id 1  end_log_pos 342 CRC32 0x07e55113 	Query	thread_id=3499	exec_time=0	error_code=0	Xid = 160237
SET TIMESTAMP=1714894000/*!*/;
SET @@session.pseudo_thread_id=3499/*!*/;
SET @@session.foreign_key_checks=1, @@session.sql_auto_is_null=0, @@session.unique_checks=1, @@session.autocommit=1/*!*/;
SET @@session.sql_mode=1168113696/*!*/;
SET @@session.auto_increment_increment=1, @@session.auto_increment_offset=1/*!*/;
/*!\C utf8mb4 *//*!*/;
SET @@session.character_set_client=255,@@session.collation_connection=255,@@session.collation_server=255/*!*/;
SET @@session.lc_time_names=0/*!*/;
SET @@session.collation_database=DEFAULT/*!*/;
/*!80011 SET @@session.default_collation_for_utf8mb4=255*//*!*/;
/*!80016 SET @@session.default_table_encryption=0*//*!*/;
create database test
/*!*/;
# at 342
#240505  7:26:55 server id 1  end_log_pos 419 CRC32 0x121b930d 	GTID	last_committed=1	sequence_number=2	rbr_only=no	original_committed_timestamp=1714894015793018	immediate_commit_timestamp=1714894015793018	transaction_length=232
# original_commit_timestamp=1714894015793018 (2024-05-05 07:26:55.793018 UTC)
# immediate_commit_timestamp=1714894015793018 (2024-05-05 07:26:55.793018 UTC)
/*!80001 SET @@session.original_commit_timestamp=1714894015793018*//*!*/;
/*!80014 SET @@session.original_server_version=80036*//*!*/;
/*!80014 SET @@session.immediate_server_version=80036*//*!*/;
SET @@SESSION.GTID_NEXT= '7f4be0b0-09b9-11ef-9bc6-0242ac130002:2'/*!*/;
# at 419
#240505  7:26:55 server id 1  end_log_pos 574 CRC32 0x71ab62bd 	Query	thread_id=3499	exec_time=0	error_code=0	Xid = 160242
SET TIMESTAMP=1714894015/*!*/;
/*!80016 SET @@session.default_table_encryption=0*//*!*/;
CREATE DATABASE IF NOT EXISTS mysqlbinlogsync
/*!*/;
# at 574
#240505  7:27:00 server id 1  end_log_pos 653 CRC32 0xd19eaea6 	GTID	last_committed=2	sequence_number=3	rbr_only=no	original_committed_timestamp=1714894020627724	immediate_commit_timestamp=1714894020627724	transaction_length=265
# original_commit_timestamp=1714894020627724 (2024-05-05 07:27:00.627724 UTC)
# immediate_commit_timestamp=1714894020627724 (2024-05-05 07:27:00.627724 UTC)
/*!80001 SET @@session.original_commit_timestamp=1714894020627724*//*!*/;
/*!80014 SET @@session.original_server_version=80036*//*!*/;
/*!80014 SET @@session.immediate_server_version=80036*//*!*/;
SET @@SESSION.GTID_NEXT= '7f4be0b0-09b9-11ef-9bc6-0242ac130002:3'/*!*/;
# at 653
#240505  7:27:00 server id 1  end_log_pos 839 CRC32 0xd29dbac8 	Query	thread_id=3499	exec_time=0	error_code=0	Xid = 160243
use `test`/*!*/;
SET TIMESTAMP=1714894020/*!*/;
/*!80013 SET @@session.sql_require_primary_key=0*//*!*/;
CREATE TABLE IF NOT EXISTS mysqlbinlogsync.sync_table (id INT PRIMARY KEY, ts DATETIME)
/*!*/;
# at 839
#240505  7:27:18 server id 1  end_log_pos 918 CRC32 0x96146c2b 	GTID	last_committed=3	sequence_number=4	rbr_only=yes	original_committed_timestamp=1714894038745041	immediate_commit_timestamp=1714894038745041	transaction_length=383
/*!50718 SET TRANSACTION ISOLATION LEVEL READ COMMITTED*//*!*/;
# original_commit_timestamp=1714894038745041 (2024-05-05 07:27:18.745041 UTC)
# immediate_commit_timestamp=1714894038745041 (2024-05-05 07:27:18.745041 UTC)
/*!80001 SET @@session.original_commit_timestamp=1714894038745041*//*!*/;
/*!80014 SET @@session.original_server_version=80036*//*!*/;
/*!80014 SET @@session.immediate_server_version=80036*//*!*/;
SET @@SESSION.GTID_NEXT= '7f4be0b0-09b9-11ef-9bc6-0242ac130002:4'/*!*/;
# at 918
#240505  7:27:18 server id 1  end_log_pos 997 CRC32 0x088d54aa 	Query	thread_id=3500	exec_time=0	error_code=0
SET TIMESTAMP=1714894038/*!*/;
SET @@session.time_zone='SYSTEM'/*!*/;
BEGIN
/*!*/;
# at 997
#240505  7:27:18 server id 1  end_log_pos 1077 CRC32 0x599f433c 	Rows_query
# REPLACE INTO mysqlbinlogsync.sync_table VALUES(1, now())
# at 1077
#240505  7:27:18 server id 1  end_log_pos 1146 CRC32 0xa725a4c3 	Table_map: `mysqlbinlogsync`.`sync_table` mapped to number 668
# has_generated_invisible_primary_key=0
# Columns(INT NOT NULL,
#         DATETIME)
ROLLBACK /* added by mysqlbinlog */ /*!*/;
SET @@SESSION.GTID_NEXT= 'AUTOMATIC' /* added by mysqlbinlog */ /*!*/;
DELIMITER ;
# End of log file
/*!50003 SET COMPLETION_TYPE=@OLD_COMPLETION_TYPE*/;
/*!50530 SET @@SESSION.PSEUDO_SLAVE_MODE=0*/;
/*!50700 SET @@SESSION.RBR_EXEC_MODE=STRICT*/;
