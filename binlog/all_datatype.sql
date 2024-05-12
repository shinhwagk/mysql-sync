# The proper term is pseudo_replica_mode, but we use this compatibility alias
# to make the statement usable on server versions 8.0.24 and older.
/*!50530 SET @@SESSION.PSEUDO_SLAVE_MODE=1*/;
/*!50003 SET @OLD_COMPLETION_TYPE=@@COMPLETION_TYPE,COMPLETION_TYPE=0*/;
/*!50700 SET @@SESSION.RBR_EXEC_MODE=IDEMPOTENT*/;

DELIMITER /*!*/;
# at 4
#700101  0:00:00 server id 1  end_log_pos 0 	Rotate to mysql-bin.000001  pos: 4
# at 4
#240512  8:56:10 server id 1  end_log_pos 126 CRC32 0xc954680f 	Start: binlog v 4, server v 8.0.36 created 240512  8:56:10 at startup
ROLLBACK/*!*/;
BINLOG '
KoRAZg8BAAAAegAAAH4AAAAAAAQAOC4wLjM2AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAqhEBmEwANAAgAAAAABAAEAAAAYgAEGggAAAAICAgCAAAACgoKKioAEjQA
CigAAQ9oVMk=
'/*!*/;
# at 126
#240512  8:56:10 server id 1  end_log_pos 157 CRC32 0x1d88746f 	Previous-GTIDs
# [empty]
# at 157
#240512  8:56:25 server id 1  end_log_pos 234 CRC32 0x99ecaa01 	GTID	last_committed=0	sequence_number=1	rbr_only=no	original_committed_timestamp=1715504185834972	immediate_commit_timestamp=1715504185834972	transaction_length=185
# original_commit_timestamp=1715504185834972 (2024-05-12 08:56:25.834972 UTC)
# immediate_commit_timestamp=1715504185834972 (2024-05-12 08:56:25.834972 UTC)
/*!80001 SET @@session.original_commit_timestamp=1715504185834972*//*!*/;
/*!80014 SET @@session.original_server_version=80036*//*!*/;
/*!80014 SET @@session.immediate_server_version=80036*//*!*/;
SET @@SESSION.GTID_NEXT= '7f4be0b0-09b9-11ef-9bc6-0242ac130002:1'/*!*/;
# at 234
#240512  8:56:25 server id 1  end_log_pos 342 CRC32 0xe3e8c14e 	Query	thread_id=45	exec_time=0	error_code=0	Xid = 133
SET TIMESTAMP=1715504185/*!*/;
SET @@session.pseudo_thread_id=45/*!*/;
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
#240512  8:56:26 server id 1  end_log_pos 421 CRC32 0xbce902a4 	GTID	last_committed=1	sequence_number=2	rbr_only=no	original_committed_timestamp=1715504186457612	immediate_commit_timestamp=1715504186457612	transaction_length=891
# original_commit_timestamp=1715504186457612 (2024-05-12 08:56:26.457612 UTC)
# immediate_commit_timestamp=1715504186457612 (2024-05-12 08:56:26.457612 UTC)
/*!80001 SET @@session.original_commit_timestamp=1715504186457612*//*!*/;
/*!80014 SET @@session.original_server_version=80036*//*!*/;
/*!80014 SET @@session.immediate_server_version=80036*//*!*/;
SET @@SESSION.GTID_NEXT= '7f4be0b0-09b9-11ef-9bc6-0242ac130002:2'/*!*/;
# at 421
#240512  8:56:26 server id 1  end_log_pos 1233 CRC32 0x7970173c 	Query	thread_id=45	exec_time=0	error_code=0	Xid = 134
SET TIMESTAMP=1715504186/*!*/;
SET @@session.explicit_defaults_for_timestamp=1/*!*/;
/*!80013 SET @@session.sql_require_primary_key=0*//*!*/;
CREATE TABLE test.data_types_demo (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `varchar_col` VARCHAR(255),     
  `char_col` CHAR(10),            
  `int_col` INT,                  
  `smallint_col` SMALLINT,        
  `tinyint_col` TINYINT,          
  `bigint_col` BIGINT,            
  `float_col` FLOAT,              
  `double_col` DOUBLE,            
  `decimal_col` DECIMAL(10, 2),   
  `date_col` DATE,                
  `time_col` TIME,                
  `datetime_col` DATETIME,        
  `timestamp_col` TIMESTAMP,      
  `year_col` YEAR,                
  `blob_col` BLOB,                
  `text_col` TEXT,                
  `enum_col` ENUM('val1', 'val2', 'val3'),  
  `set_col` SET('set1', 'set2', 'set3')     
)
/*!*/;
# at 1233
#240512  8:56:26 server id 1  end_log_pos 1312 CRC32 0xf6b749e5 	GTID	last_committed=2	sequence_number=3	rbr_only=yes	original_committed_timestamp=1715504186578911	immediate_commit_timestamp=1715504186578911	transaction_length=1522
/*!50718 SET TRANSACTION ISOLATION LEVEL READ COMMITTED*//*!*/;
# original_commit_timestamp=1715504186578911 (2024-05-12 08:56:26.578911 UTC)
# immediate_commit_timestamp=1715504186578911 (2024-05-12 08:56:26.578911 UTC)
/*!80001 SET @@session.original_commit_timestamp=1715504186578911*//*!*/;
/*!80014 SET @@session.original_server_version=80036*//*!*/;
/*!80014 SET @@session.immediate_server_version=80036*//*!*/;
SET @@SESSION.GTID_NEXT= '7f4be0b0-09b9-11ef-9bc6-0242ac130002:3'/*!*/;
# at 1312
#240512  8:56:26 server id 1  end_log_pos 1391 CRC32 0x9752d085 	Query	thread_id=45	exec_time=0	error_code=0
SET TIMESTAMP=1715504186/*!*/;
SET @@session.time_zone='SYSTEM'/*!*/;
BEGIN
/*!*/;
# at 1391
#240512  8:56:26 server id 1  end_log_pos 2223 CRC32 0x78b54526 	Rows_query
# INSERT INTO test.data_types_demo (
#   varchar_col, 
#   char_col, 
#   int_col, 
#   smallint_col, 
#   tinyint_col, 
#   bigint_col, 
#   float_col, 
#   double_col, 
#   decimal_col, 
#   date_col, 
#   time_col, 
#   datetime_col, 
#   timestamp_col, 
#   year_col, 
#   blob_col, 
#   text_col, 
#   enum_col, 
#   set_col
# ) VALUES (
#   'Example text',          
#   'ABCDE',                 
#   12345,                   
#   32767,                   
#   127,                     
#   9223372036854775807,     
#   12345.678,               
#   12345678.91011,          
#   12345.67,                
#   '2024-05-10',            
#   '15:30:00',              
#   '2024-05-10 15:30:00',   
#   CURRENT_TIMESTAMP,       
#   2024,                    
#   'binary data here',      
#   'Longer piece of text',  
#   'val1',                  
#   'set1,set2'              
# )
# at 2223
#240512  8:56:26 server id 1  end_log_pos 2570 CRC32 0x42c173b4 	Table_map: `test`.`data_types_demo` mapped to number 117
# has_generated_invisible_primary_key=0
# Columns(`id` INT NOT NULL,
#         `varchar_col` VARCHAR(255) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `char_col` CHAR(10) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `int_col` INT,
#         `smallint_col` SMALLINT,
#         `tinyint_col` TINYINT,
#         `bigint_col` BIGINT,
#         `float_col` FLOAT,
#         `double_col` DOUBLE,
#         `decimal_col` DECIMAL(10,2),
#         `date_col` DATE,
#         `time_col` TIME,
#         `datetime_col` DATETIME,
#         `timestamp_col` TIMESTAMP,
#         `year_col` YEAR,
#         `blob_col` BLOB,
#         `text_col` TEXT CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `enum_col` ENUM('val1', 'val2', 'val3') CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `set_col` SET('set1', 'set2', 'set3') CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci)
# Primary Key(id)
# at 2570
#240512  8:56:26 server id 1  end_log_pos 2724 CRC32 0x6854b791 	Write_rows: table id 117 flags: STMT_END_F

BINLOG '
OoRAZh0BAAAAQAMAAK8IAACAAChJTlNFUlQgSU5UTyB0ZXN0LmRhdGFfdHlwZXNfZGVtbyAoCiAg
dmFyY2hhcl9jb2wsIAogIGNoYXJfY29sLCAKICBpbnRfY29sLCAKICBzbWFsbGludF9jb2wsIAog
IHRpbnlpbnRfY29sLCAKICBiaWdpbnRfY29sLCAKICBmbG9hdF9jb2wsIAogIGRvdWJsZV9jb2ws
IAogIGRlY2ltYWxfY29sLCAKICBkYXRlX2NvbCwgCiAgdGltZV9jb2wsIAogIGRhdGV0aW1lX2Nv
bCwgCiAgdGltZXN0YW1wX2NvbCwgCiAgeWVhcl9jb2wsIAogIGJsb2JfY29sLCAKICB0ZXh0X2Nv
bCwgCiAgZW51bV9jb2wsIAogIHNldF9jb2wKKSBWQUxVRVMgKAogICdFeGFtcGxlIHRleHQnLCAg
ICAgICAgICAKICAnQUJDREUnLCAgICAgICAgICAgICAgICAgCiAgMTIzNDUsICAgICAgICAgICAg
ICAgICAgIAogIDMyNzY3LCAgICAgICAgICAgICAgICAgICAKICAxMjcsICAgICAgICAgICAgICAg
ICAgICAgCiAgOTIyMzM3MjAzNjg1NDc3NTgwNywgICAgIAogIDEyMzQ1LjY3OCwgICAgICAgICAg
ICAgICAKICAxMjM0NTY3OC45MTAxMSwgICAgICAgICAgCiAgMTIzNDUuNjcsICAgICAgICAgICAg
ICAgIAogICcyMDI0LTA1LTEwJywgICAgICAgICAgICAKICAnMTU6MzA6MDAnLCAgICAgICAgICAg
ICAgCiAgJzIwMjQtMDUtMTAgMTU6MzA6MDAnLCAgIAogIENVUlJFTlRfVElNRVNUQU1QLCAgICAg
ICAKICAyMDI0LCAgICAgICAgICAgICAgICAgICAgCiAgJ2JpbmFyeSBkYXRhIGhlcmUnLCAgICAg
IAogICdMb25nZXIgcGllY2Ugb2YgdGV4dCcsICAKICAndmFsMScsICAgICAgICAgICAgICAgICAg
CiAgJ3NldDEsc2V0MicgICAgICAgICAgICAgIAopJkW1eA==
OoRAZhMBAAAAWwEAAAoKAAAAAHUAAAAAAAEABHRlc3QAD2RhdGFfdHlwZXNfZGVtbwATAw/+AwIB
CAQF9goTEhEN/Pz+/hH8A/4oBAgKAgAAAAIC9wH4Af7/BwECAIACBfz/AAI/BL4CaWQLdmFyY2hh
cl9jb2wIY2hhcl9jb2wHaW50X2NvbAxzbWFsbGludF9jb2wLdGlueWludF9jb2wKYmlnaW50X2Nv
bAlmbG9hdF9jb2wKZG91YmxlX2NvbAtkZWNpbWFsX2NvbAhkYXRlX2NvbAh0aW1lX2NvbAxkYXRl
dGltZV9jb2wNdGltZXN0YW1wX2NvbAh5ZWFyX2NvbAhibG9iX2NvbAh0ZXh0X2NvbAhlbnVtX2Nv
bAdzZXRfY29sCgP8/wAFEAMEc2V0MQRzZXQyBHNldDMGEAMEdmFsMQR2YWwyBHZhbDMIAQAMA///
4LRzwUI=
OoRAZh4BAAAAmgAAAKQKAAAAAHUAAAAAAAEAAgAT////AAAAAQAAAAwARXhhbXBsZSB0ZXh0BUFC
Q0RFOTAAAP9/f/////////9/tuZARgKfH90pjGdBgAAwOUOq0A+A94CZs1T3gGZAhDp8EABiaW5h
cnkgZGF0YSBoZXJlFABMb25nZXIgcGllY2Ugb2YgdGV4dAEDkbdUaA==
'/*!*/;
### INSERT INTO `test`.`data_types_demo`
### SET
###   @1=1 /* INT meta=0 nullable=0 is_null=0 */
###   @2='Example text' /* VARSTRING(1020) meta=1020 nullable=1 is_null=0 */
###   @3='ABCDE' /* STRING(40) meta=65064 nullable=1 is_null=0 */
###   @4=12345 /* INT meta=0 nullable=1 is_null=0 */
###   @5=32767 /* SHORTINT meta=0 nullable=1 is_null=0 */
###   @6=127 /* TINYINT meta=0 nullable=1 is_null=0 */
###   @7=9223372036854775807 /* LONGINT meta=0 nullable=1 is_null=0 */
###   @8=12345.7              /* FLOAT meta=4 nullable=1 is_null=0 */
###   @9=12345678.910110000521 /* DOUBLE meta=8 nullable=1 is_null=0 */
###   @10=12345.67 /* DECIMAL(10,2) meta=2562 nullable=1 is_null=0 */
###   @11='2024:05:10' /* DATE meta=0 nullable=1 is_null=0 */
###   @12='15:30:00' /* TIME(0) meta=0 nullable=1 is_null=0 */
###   @13='2024-05-10 15:30:00' /* DATETIME(0) meta=0 nullable=1 is_null=0 */
###   @14=1715504186 /* TIMESTAMP(0) meta=0 nullable=1 is_null=0 */
###   @15=2024 /* YEAR meta=0 nullable=1 is_null=0 */
###   @16='binary data here' /* BLOB/TEXT meta=2 nullable=1 is_null=0 */
###   @17='Longer piece of text' /* BLOB/TEXT meta=2 nullable=1 is_null=0 */
###   @18=1 /* ENUM(1 byte) meta=63233 nullable=1 is_null=0 */
###   @19=b'00000011' /* SET(1 bytes) meta=63489 nullable=1 is_null=0 */
# at 2724
#240512  8:56:26 server id 1  end_log_pos 2755 CRC32 0x5fba112a 	Xid = 135
COMMIT/*!*/;
# at 2755
#240512  8:59:45 server id 1  end_log_pos 2834 CRC32 0xbf7a6a9f 	GTID	last_committed=3	sequence_number=4	rbr_only=yes	original_committed_timestamp=1715504385744810	immediate_commit_timestamp=1715504385744810	transaction_length=890
/*!50718 SET TRANSACTION ISOLATION LEVEL READ COMMITTED*//*!*/;
# original_commit_timestamp=1715504385744810 (2024-05-12 08:59:45.744810 UTC)
# immediate_commit_timestamp=1715504385744810 (2024-05-12 08:59:45.744810 UTC)
/*!80001 SET @@session.original_commit_timestamp=1715504385744810*//*!*/;
/*!80014 SET @@session.original_server_version=80036*//*!*/;
/*!80014 SET @@session.immediate_server_version=80036*//*!*/;
SET @@SESSION.GTID_NEXT= '7f4be0b0-09b9-11ef-9bc6-0242ac130002:4'/*!*/;
# at 2834
#240512  8:59:45 server id 1  end_log_pos 2914 CRC32 0x6fffd571 	Query	thread_id=47	exec_time=0	error_code=0
SET TIMESTAMP=1715504385/*!*/;
BEGIN
/*!*/;
# at 2914
#240512  8:59:45 server id 1  end_log_pos 2993 CRC32 0x61f7ef49 	Rows_query
# update test.data_types_demo set int_col=2222 where id=1
# at 2993
#240512  8:59:45 server id 1  end_log_pos 3340 CRC32 0xc888d776 	Table_map: `test`.`data_types_demo` mapped to number 117
# has_generated_invisible_primary_key=0
# Columns(`id` INT NOT NULL,
#         `varchar_col` VARCHAR(255) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `char_col` CHAR(10) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `int_col` INT,
#         `smallint_col` SMALLINT,
#         `tinyint_col` TINYINT,
#         `bigint_col` BIGINT,
#         `float_col` FLOAT,
#         `double_col` DOUBLE,
#         `decimal_col` DECIMAL(10,2),
#         `date_col` DATE,
#         `time_col` TIME,
#         `datetime_col` DATETIME,
#         `timestamp_col` TIMESTAMP,
#         `year_col` YEAR,
#         `blob_col` BLOB,
#         `text_col` TEXT CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `enum_col` ENUM('val1', 'val2', 'val3') CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `set_col` SET('set1', 'set2', 'set3') CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci)
# Primary Key(id)
# at 3340
#240512  8:59:45 server id 1  end_log_pos 3614 CRC32 0x6ffecbbc 	Update_rows: table id 117 flags: STMT_END_F

BINLOG '
AYVAZh0BAAAATwAAALELAACAADd1cGRhdGUgdGVzdC5kYXRhX3R5cGVzX2RlbW8gc2V0IGludF9j
b2w9MjIyMiB3aGVyZSBpZD0xSe/3YQ==
AYVAZhMBAAAAWwEAAAwNAAAAAHUAAAAAAAEABHRlc3QAD2RhdGFfdHlwZXNfZGVtbwATAw/+AwIB
CAQF9goTEhEN/Pz+/hH8A/4oBAgKAgAAAAIC9wH4Af7/BwECAIACBfz/AAI/BL4CaWQLdmFyY2hh
cl9jb2wIY2hhcl9jb2wHaW50X2NvbAxzbWFsbGludF9jb2wLdGlueWludF9jb2wKYmlnaW50X2Nv
bAlmbG9hdF9jb2wKZG91YmxlX2NvbAtkZWNpbWFsX2NvbAhkYXRlX2NvbAh0aW1lX2NvbAxkYXRl
dGltZV9jb2wNdGltZXN0YW1wX2NvbAh5ZWFyX2NvbAhibG9iX2NvbAh0ZXh0X2NvbAhlbnVtX2Nv
bAdzZXRfY29sCgP8/wAFEAMEc2V0MQRzZXQyBHNldDMGEAMEdmFsMQR2YWwyBHZhbDMIAQAMA///
4HbXiMg=
AYVAZh8BAAAAEgEAAB4OAAAAAHUAAAAAAAEAAgAT////////AAAAAQAAAAwARXhhbXBsZSB0ZXh0
BUFCQ0RFOTAAAP9/f/////////9/tuZARgKfH90pjGdBgAAwOUOq0A+A94CZs1T3gGZAhDp8EABi
aW5hcnkgZGF0YSBoZXJlFABMb25nZXIgcGllY2Ugb2YgdGV4dAEDAAAAAQAAAAwARXhhbXBsZSB0
ZXh0BUFCQ0RFrggAAP9/f/////////9/tuZARgKfH90pjGdBgAAwOUOq0A+A94CZs1T3gGZAhDp8
EABiaW5hcnkgZGF0YSBoZXJlFABMb25nZXIgcGllY2Ugb2YgdGV4dAEDvMv+bw==
'/*!*/;
### UPDATE `test`.`data_types_demo`
### WHERE
###   @1=1 /* INT meta=0 nullable=0 is_null=0 */
###   @2='Example text' /* VARSTRING(1020) meta=1020 nullable=1 is_null=0 */
###   @3='ABCDE' /* STRING(40) meta=65064 nullable=1 is_null=0 */
###   @4=12345 /* INT meta=0 nullable=1 is_null=0 */
###   @5=32767 /* SHORTINT meta=0 nullable=1 is_null=0 */
###   @6=127 /* TINYINT meta=0 nullable=1 is_null=0 */
###   @7=9223372036854775807 /* LONGINT meta=0 nullable=1 is_null=0 */
###   @8=12345.7              /* FLOAT meta=4 nullable=1 is_null=0 */
###   @9=12345678.910110000521 /* DOUBLE meta=8 nullable=1 is_null=0 */
###   @10=12345.67 /* DECIMAL(10,2) meta=2562 nullable=1 is_null=0 */
###   @11='2024:05:10' /* DATE meta=0 nullable=1 is_null=0 */
###   @12='15:30:00' /* TIME(0) meta=0 nullable=1 is_null=0 */
###   @13='2024-05-10 15:30:00' /* DATETIME(0) meta=0 nullable=1 is_null=0 */
###   @14=1715504186 /* TIMESTAMP(0) meta=0 nullable=1 is_null=0 */
###   @15=2024 /* YEAR meta=0 nullable=1 is_null=0 */
###   @16='binary data here' /* BLOB/TEXT meta=2 nullable=1 is_null=0 */
###   @17='Longer piece of text' /* BLOB/TEXT meta=2 nullable=1 is_null=0 */
###   @18=1 /* ENUM(1 byte) meta=63233 nullable=1 is_null=0 */
###   @19=b'00000011' /* SET(1 bytes) meta=63489 nullable=1 is_null=0 */
### SET
###   @1=1 /* INT meta=0 nullable=0 is_null=0 */
###   @2='Example text' /* VARSTRING(1020) meta=1020 nullable=1 is_null=0 */
###   @3='ABCDE' /* STRING(40) meta=65064 nullable=1 is_null=0 */
###   @4=2222 /* INT meta=0 nullable=1 is_null=0 */
###   @5=32767 /* SHORTINT meta=0 nullable=1 is_null=0 */
###   @6=127 /* TINYINT meta=0 nullable=1 is_null=0 */
###   @7=9223372036854775807 /* LONGINT meta=0 nullable=1 is_null=0 */
###   @8=12345.7              /* FLOAT meta=4 nullable=1 is_null=0 */
###   @9=12345678.910110000521 /* DOUBLE meta=8 nullable=1 is_null=0 */
###   @10=12345.67 /* DECIMAL(10,2) meta=2562 nullable=1 is_null=0 */
###   @11='2024:05:10' /* DATE meta=0 nullable=1 is_null=0 */
###   @12='15:30:00' /* TIME(0) meta=0 nullable=1 is_null=0 */
###   @13='2024-05-10 15:30:00' /* DATETIME(0) meta=0 nullable=1 is_null=0 */
###   @14=1715504186 /* TIMESTAMP(0) meta=0 nullable=1 is_null=0 */
###   @15=2024 /* YEAR meta=0 nullable=1 is_null=0 */
###   @16='binary data here' /* BLOB/TEXT meta=2 nullable=1 is_null=0 */
###   @17='Longer piece of text' /* BLOB/TEXT meta=2 nullable=1 is_null=0 */
###   @18=1 /* ENUM(1 byte) meta=63233 nullable=1 is_null=0 */
###   @19=b'00000011' /* SET(1 bytes) meta=63489 nullable=1 is_null=0 */
# at 3614
#240512  8:59:45 server id 1  end_log_pos 3645 CRC32 0xd8d2ce72 	Xid = 141
COMMIT/*!*/;
# at 3645
#240512  8:59:45 server id 1  end_log_pos 3724 CRC32 0xc8f781cb 	GTID	last_committed=4	sequence_number=5	rbr_only=yes	original_committed_timestamp=1715504385865868	immediate_commit_timestamp=1715504385865868	transaction_length=750
/*!50718 SET TRANSACTION ISOLATION LEVEL READ COMMITTED*//*!*/;
# original_commit_timestamp=1715504385865868 (2024-05-12 08:59:45.865868 UTC)
# immediate_commit_timestamp=1715504385865868 (2024-05-12 08:59:45.865868 UTC)
/*!80001 SET @@session.original_commit_timestamp=1715504385865868*//*!*/;
/*!80014 SET @@session.original_server_version=80036*//*!*/;
/*!80014 SET @@session.immediate_server_version=80036*//*!*/;
SET @@SESSION.GTID_NEXT= '7f4be0b0-09b9-11ef-9bc6-0242ac130002:5'/*!*/;
# at 3724
#240512  8:59:45 server id 1  end_log_pos 3795 CRC32 0x3ed657ee 	Query	thread_id=47	exec_time=0	error_code=0
SET TIMESTAMP=1715504385/*!*/;
BEGIN
/*!*/;
# at 3795
#240512  8:59:45 server id 1  end_log_pos 3863 CRC32 0x325b6297 	Rows_query
# delete from test.data_types_demo  where id=1
# at 3863
#240512  8:59:45 server id 1  end_log_pos 4210 CRC32 0xd11a7195 	Table_map: `test`.`data_types_demo` mapped to number 117
# has_generated_invisible_primary_key=0
# Columns(`id` INT NOT NULL,
#         `varchar_col` VARCHAR(255) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `char_col` CHAR(10) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `int_col` INT,
#         `smallint_col` SMALLINT,
#         `tinyint_col` TINYINT,
#         `bigint_col` BIGINT,
#         `float_col` FLOAT,
#         `double_col` DOUBLE,
#         `decimal_col` DECIMAL(10,2),
#         `date_col` DATE,
#         `time_col` TIME,
#         `datetime_col` DATETIME,
#         `timestamp_col` TIMESTAMP,
#         `year_col` YEAR,
#         `blob_col` BLOB,
#         `text_col` TEXT CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `enum_col` ENUM('val1', 'val2', 'val3') CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `set_col` SET('set1', 'set2', 'set3') CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci)
# Primary Key(id)
# at 4210
#240512  8:59:45 server id 1  end_log_pos 4364 CRC32 0x8209164d 	Delete_rows: table id 117 flags: STMT_END_F

BINLOG '
AYVAZh0BAAAARAAAABcPAACAACxkZWxldGUgZnJvbSB0ZXN0LmRhdGFfdHlwZXNfZGVtbyAgd2hl
cmUgaWQ9MZdiWzI=
AYVAZhMBAAAAWwEAAHIQAAAAAHUAAAAAAAEABHRlc3QAD2RhdGFfdHlwZXNfZGVtbwATAw/+AwIB
CAQF9goTEhEN/Pz+/hH8A/4oBAgKAgAAAAIC9wH4Af7/BwECAIACBfz/AAI/BL4CaWQLdmFyY2hh
cl9jb2wIY2hhcl9jb2wHaW50X2NvbAxzbWFsbGludF9jb2wLdGlueWludF9jb2wKYmlnaW50X2Nv
bAlmbG9hdF9jb2wKZG91YmxlX2NvbAtkZWNpbWFsX2NvbAhkYXRlX2NvbAh0aW1lX2NvbAxkYXRl
dGltZV9jb2wNdGltZXN0YW1wX2NvbAh5ZWFyX2NvbAhibG9iX2NvbAh0ZXh0X2NvbAhlbnVtX2Nv
bAdzZXRfY29sCgP8/wAFEAMEc2V0MQRzZXQyBHNldDMGEAMEdmFsMQR2YWwyBHZhbDMIAQAMA///
4JVxGtE=
AYVAZiABAAAAmgAAAAwRAAAAAHUAAAAAAAEAAgAT////AAAAAQAAAAwARXhhbXBsZSB0ZXh0BUFC
Q0RFrggAAP9/f/////////9/tuZARgKfH90pjGdBgAAwOUOq0A+A94CZs1T3gGZAhDp8EABiaW5h
cnkgZGF0YSBoZXJlFABMb25nZXIgcGllY2Ugb2YgdGV4dAEDTRYJgg==
'/*!*/;
### DELETE FROM `test`.`data_types_demo`
### WHERE
###   @1=1 /* INT meta=0 nullable=0 is_null=0 */
###   @2='Example text' /* VARSTRING(1020) meta=1020 nullable=1 is_null=0 */
###   @3='ABCDE' /* STRING(40) meta=65064 nullable=1 is_null=0 */
###   @4=2222 /* INT meta=0 nullable=1 is_null=0 */
###   @5=32767 /* SHORTINT meta=0 nullable=1 is_null=0 */
###   @6=127 /* TINYINT meta=0 nullable=1 is_null=0 */
###   @7=9223372036854775807 /* LONGINT meta=0 nullable=1 is_null=0 */
###   @8=12345.7              /* FLOAT meta=4 nullable=1 is_null=0 */
###   @9=12345678.910110000521 /* DOUBLE meta=8 nullable=1 is_null=0 */
###   @10=12345.67 /* DECIMAL(10,2) meta=2562 nullable=1 is_null=0 */
###   @11='2024:05:10' /* DATE meta=0 nullable=1 is_null=0 */
###   @12='15:30:00' /* TIME(0) meta=0 nullable=1 is_null=0 */
###   @13='2024-05-10 15:30:00' /* DATETIME(0) meta=0 nullable=1 is_null=0 */
###   @14=1715504186 /* TIMESTAMP(0) meta=0 nullable=1 is_null=0 */
###   @15=2024 /* YEAR meta=0 nullable=1 is_null=0 */
###   @16='binary data here' /* BLOB/TEXT meta=2 nullable=1 is_null=0 */
###   @17='Longer piece of text' /* BLOB/TEXT meta=2 nullable=1 is_null=0 */
###   @18=1 /* ENUM(1 byte) meta=63233 nullable=1 is_null=0 */
###   @19=b'00000011' /* SET(1 bytes) meta=63489 nullable=1 is_null=0 */
# at 4364
#240512  8:59:45 server id 1  end_log_pos 4395 CRC32 0x5b5acda4 	Xid = 142
COMMIT/*!*/;
# at 4395
#240512  9:09:25 server id 1  end_log_pos 4472 CRC32 0x5473364e 	GTID	last_committed=5	sequence_number=6	rbr_only=no	original_committed_timestamp=1715504965949328	immediate_commit_timestamp=1715504965949328	transaction_length=225
# original_commit_timestamp=1715504965949328 (2024-05-12 09:09:25.949328 UTC)
# immediate_commit_timestamp=1715504965949328 (2024-05-12 09:09:25.949328 UTC)
/*!80001 SET @@session.original_commit_timestamp=1715504965949328*//*!*/;
/*!80014 SET @@session.original_server_version=80036*//*!*/;
/*!80014 SET @@session.immediate_server_version=80036*//*!*/;
SET @@SESSION.GTID_NEXT= '7f4be0b0-09b9-11ef-9bc6-0242ac130002:6'/*!*/;
# at 4472
#240512  9:09:25 server id 1  end_log_pos 4620 CRC32 0xe0905a82 	Query	thread_id=49	exec_time=0	error_code=0	Xid = 148
SET TIMESTAMP=1715504965/*!*/;
/*!80013 SET @@session.sql_require_primary_key=0*//*!*/;
alter table test.data_types_demo add index idx_int_col (int_col)
/*!*/;
# at 4620
#240512  9:09:59 server id 1  end_log_pos 4699 CRC32 0xe715fbda 	GTID	last_committed=6	sequence_number=7	rbr_only=yes	original_committed_timestamp=1715504999139897	immediate_commit_timestamp=1715504999139897	transaction_length=1522
/*!50718 SET TRANSACTION ISOLATION LEVEL READ COMMITTED*//*!*/;
# original_commit_timestamp=1715504999139897 (2024-05-12 09:09:59.139897 UTC)
# immediate_commit_timestamp=1715504999139897 (2024-05-12 09:09:59.139897 UTC)
/*!80001 SET @@session.original_commit_timestamp=1715504999139897*//*!*/;
/*!80014 SET @@session.original_server_version=80036*//*!*/;
/*!80014 SET @@session.immediate_server_version=80036*//*!*/;
SET @@SESSION.GTID_NEXT= '7f4be0b0-09b9-11ef-9bc6-0242ac130002:7'/*!*/;
# at 4699
#240512  9:09:59 server id 1  end_log_pos 4778 CRC32 0xdaff4f91 	Query	thread_id=51	exec_time=0	error_code=0
SET TIMESTAMP=1715504999/*!*/;
BEGIN
/*!*/;
# at 4778
#240512  9:09:59 server id 1  end_log_pos 5610 CRC32 0x00a9e150 	Rows_query
# INSERT INTO test.data_types_demo (
#   varchar_col, 
#   char_col, 
#   int_col, 
#   smallint_col, 
#   tinyint_col, 
#   bigint_col, 
#   float_col, 
#   double_col, 
#   decimal_col, 
#   date_col, 
#   time_col, 
#   datetime_col, 
#   timestamp_col, 
#   year_col, 
#   blob_col, 
#   text_col, 
#   enum_col, 
#   set_col
# ) VALUES (
#   'Example text',          
#   'ABCDE',                 
#   12345,                   
#   32767,                   
#   127,                     
#   9223372036854775807,     
#   12345.678,               
#   12345678.91011,          
#   12345.67,                
#   '2024-05-10',            
#   '15:30:00',              
#   '2024-05-10 15:30:00',   
#   CURRENT_TIMESTAMP,       
#   2024,                    
#   'binary data here',      
#   'Longer piece of text',  
#   'val1',                  
#   'set1,set2'              
# )
# at 5610
#240512  9:09:59 server id 1  end_log_pos 5957 CRC32 0xf14eac89 	Table_map: `test`.`data_types_demo` mapped to number 119
# has_generated_invisible_primary_key=0
# Columns(`id` INT NOT NULL,
#         `varchar_col` VARCHAR(255) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `char_col` CHAR(10) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `int_col` INT,
#         `smallint_col` SMALLINT,
#         `tinyint_col` TINYINT,
#         `bigint_col` BIGINT,
#         `float_col` FLOAT,
#         `double_col` DOUBLE,
#         `decimal_col` DECIMAL(10,2),
#         `date_col` DATE,
#         `time_col` TIME,
#         `datetime_col` DATETIME,
#         `timestamp_col` TIMESTAMP,
#         `year_col` YEAR,
#         `blob_col` BLOB,
#         `text_col` TEXT CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `enum_col` ENUM('val1', 'val2', 'val3') CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `set_col` SET('set1', 'set2', 'set3') CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci)
# Primary Key(id)
# at 5957
#240512  9:09:59 server id 1  end_log_pos 6111 CRC32 0x818a5695 	Write_rows: table id 119 flags: STMT_END_F

BINLOG '
Z4dAZh0BAAAAQAMAAOoVAACAAChJTlNFUlQgSU5UTyB0ZXN0LmRhdGFfdHlwZXNfZGVtbyAoCiAg
dmFyY2hhcl9jb2wsIAogIGNoYXJfY29sLCAKICBpbnRfY29sLCAKICBzbWFsbGludF9jb2wsIAog
IHRpbnlpbnRfY29sLCAKICBiaWdpbnRfY29sLCAKICBmbG9hdF9jb2wsIAogIGRvdWJsZV9jb2ws
IAogIGRlY2ltYWxfY29sLCAKICBkYXRlX2NvbCwgCiAgdGltZV9jb2wsIAogIGRhdGV0aW1lX2Nv
bCwgCiAgdGltZXN0YW1wX2NvbCwgCiAgeWVhcl9jb2wsIAogIGJsb2JfY29sLCAKICB0ZXh0X2Nv
bCwgCiAgZW51bV9jb2wsIAogIHNldF9jb2wKKSBWQUxVRVMgKAogICdFeGFtcGxlIHRleHQnLCAg
ICAgICAgICAKICAnQUJDREUnLCAgICAgICAgICAgICAgICAgCiAgMTIzNDUsICAgICAgICAgICAg
ICAgICAgIAogIDMyNzY3LCAgICAgICAgICAgICAgICAgICAKICAxMjcsICAgICAgICAgICAgICAg
ICAgICAgCiAgOTIyMzM3MjAzNjg1NDc3NTgwNywgICAgIAogIDEyMzQ1LjY3OCwgICAgICAgICAg
ICAgICAKICAxMjM0NTY3OC45MTAxMSwgICAgICAgICAgCiAgMTIzNDUuNjcsICAgICAgICAgICAg
ICAgIAogICcyMDI0LTA1LTEwJywgICAgICAgICAgICAKICAnMTU6MzA6MDAnLCAgICAgICAgICAg
ICAgCiAgJzIwMjQtMDUtMTAgMTU6MzA6MDAnLCAgIAogIENVUlJFTlRfVElNRVNUQU1QLCAgICAg
ICAKICAyMDI0LCAgICAgICAgICAgICAgICAgICAgCiAgJ2JpbmFyeSBkYXRhIGhlcmUnLCAgICAg
IAogICdMb25nZXIgcGllY2Ugb2YgdGV4dCcsICAKICAndmFsMScsICAgICAgICAgICAgICAgICAg
CiAgJ3NldDEsc2V0MicgICAgICAgICAgICAgIAopUOGpAA==
Z4dAZhMBAAAAWwEAAEUXAAAAAHcAAAAAAAEABHRlc3QAD2RhdGFfdHlwZXNfZGVtbwATAw/+AwIB
CAQF9goTEhEN/Pz+/hH8A/4oBAgKAgAAAAIC9wH4Af7/BwECAIACBfz/AAI/BL4CaWQLdmFyY2hh
cl9jb2wIY2hhcl9jb2wHaW50X2NvbAxzbWFsbGludF9jb2wLdGlueWludF9jb2wKYmlnaW50X2Nv
bAlmbG9hdF9jb2wKZG91YmxlX2NvbAtkZWNpbWFsX2NvbAhkYXRlX2NvbAh0aW1lX2NvbAxkYXRl
dGltZV9jb2wNdGltZXN0YW1wX2NvbAh5ZWFyX2NvbAhibG9iX2NvbAh0ZXh0X2NvbAhlbnVtX2Nv
bAdzZXRfY29sCgP8/wAFEAMEc2V0MQRzZXQyBHNldDMGEAMEdmFsMQR2YWwyBHZhbDMIAQAMA///
4ImsTvE=
Z4dAZh4BAAAAmgAAAN8XAAAAAHcAAAAAAAEAAgAT////AAAAAgAAAAwARXhhbXBsZSB0ZXh0BUFC
Q0RFOTAAAP9/f/////////9/tuZARgKfH90pjGdBgAAwOUOq0A+A94CZs1T3gGZAh2d8EABiaW5h
cnkgZGF0YSBoZXJlFABMb25nZXIgcGllY2Ugb2YgdGV4dAEDlVaKgQ==
'/*!*/;
### INSERT INTO `test`.`data_types_demo`
### SET
###   @1=2 /* INT meta=0 nullable=0 is_null=0 */
###   @2='Example text' /* VARSTRING(1020) meta=1020 nullable=1 is_null=0 */
###   @3='ABCDE' /* STRING(40) meta=65064 nullable=1 is_null=0 */
###   @4=12345 /* INT meta=0 nullable=1 is_null=0 */
###   @5=32767 /* SHORTINT meta=0 nullable=1 is_null=0 */
###   @6=127 /* TINYINT meta=0 nullable=1 is_null=0 */
###   @7=9223372036854775807 /* LONGINT meta=0 nullable=1 is_null=0 */
###   @8=12345.7              /* FLOAT meta=4 nullable=1 is_null=0 */
###   @9=12345678.910110000521 /* DOUBLE meta=8 nullable=1 is_null=0 */
###   @10=12345.67 /* DECIMAL(10,2) meta=2562 nullable=1 is_null=0 */
###   @11='2024:05:10' /* DATE meta=0 nullable=1 is_null=0 */
###   @12='15:30:00' /* TIME(0) meta=0 nullable=1 is_null=0 */
###   @13='2024-05-10 15:30:00' /* DATETIME(0) meta=0 nullable=1 is_null=0 */
###   @14=1715504999 /* TIMESTAMP(0) meta=0 nullable=1 is_null=0 */
###   @15=2024 /* YEAR meta=0 nullable=1 is_null=0 */
###   @16='binary data here' /* BLOB/TEXT meta=2 nullable=1 is_null=0 */
###   @17='Longer piece of text' /* BLOB/TEXT meta=2 nullable=1 is_null=0 */
###   @18=1 /* ENUM(1 byte) meta=63233 nullable=1 is_null=0 */
###   @19=b'00000011' /* SET(1 bytes) meta=63489 nullable=1 is_null=0 */
# at 6111
#240512  9:09:59 server id 1  end_log_pos 6142 CRC32 0x94cc8cad 	Xid = 154
COMMIT/*!*/;
# at 6142
#240512 13:09:11 server id 1  end_log_pos 6221 CRC32 0x5bf3180d 	GTID	last_committed=7	sequence_number=8	rbr_only=yes	original_committed_timestamp=1715519351489156	immediate_commit_timestamp=1715519351489156	transaction_length=1556
/*!50718 SET TRANSACTION ISOLATION LEVEL READ COMMITTED*//*!*/;
# original_commit_timestamp=1715519351489156 (2024-05-12 13:09:11.489156 UTC)
# immediate_commit_timestamp=1715519351489156 (2024-05-12 13:09:11.489156 UTC)
/*!80001 SET @@session.original_commit_timestamp=1715519351489156*//*!*/;
/*!80014 SET @@session.original_server_version=80036*//*!*/;
/*!80014 SET @@session.immediate_server_version=80036*//*!*/;
SET @@SESSION.GTID_NEXT= '7f4be0b0-09b9-11ef-9bc6-0242ac130002:8'/*!*/;
# at 6221
#240512 13:09:11 server id 1  end_log_pos 6300 CRC32 0x27fd2d84 	Query	thread_id=58	exec_time=0	error_code=0
SET TIMESTAMP=1715519351/*!*/;
BEGIN
/*!*/;
# at 6300
#240512 13:09:11 server id 1  end_log_pos 7166 CRC32 0x7842797a 	Rows_query
# INSERT INTO test.data_types_demo (
#    varchar_col, 
#    char_col, 
#    int_col, 
#    smallint_col, 
#    tinyint_col, 
#    bigint_col, 
#    float_col, 
#    double_col, 
#    decimal_col, 
#    date_col, 
#    time_col, 
#    datetime_col, 
#    timestamp_col, 
#    year_col, 
#    blob_col, 
#    text_col, 
#    enum_col, 
#    set_col
#  ) VALUES (
#    'Example text',          
#    'ABCDE',                 
#    12345,                   
#    32767,                   
#    127,                     
#    9223372036854775807,     
#    12345.678,               
#    12345678.91011,          
#    12345.67,                
#    '2024-05-10',            
#    '15:30:00',              
#    '2024-05-10 15:30:00',   
#    CURRENT_TIMESTAMP,       
#    2024,                    
#    'binary data here',      
#    'Longer piece of text',  
#    'val1',                  
#    'set1 '              
#  )
# at 7166
#240512 13:09:11 server id 1  end_log_pos 7513 CRC32 0xe9b51922 	Table_map: `test`.`data_types_demo` mapped to number 119
# has_generated_invisible_primary_key=0
# Columns(`id` INT NOT NULL,
#         `varchar_col` VARCHAR(255) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `char_col` CHAR(10) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `int_col` INT,
#         `smallint_col` SMALLINT,
#         `tinyint_col` TINYINT,
#         `bigint_col` BIGINT,
#         `float_col` FLOAT,
#         `double_col` DOUBLE,
#         `decimal_col` DECIMAL(10,2),
#         `date_col` DATE,
#         `time_col` TIME,
#         `datetime_col` DATETIME,
#         `timestamp_col` TIMESTAMP,
#         `year_col` YEAR,
#         `blob_col` BLOB,
#         `text_col` TEXT CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `enum_col` ENUM('val1', 'val2', 'val3') CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `set_col` SET('set1', 'set2', 'set3') CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci)
# Primary Key(id)
# at 7513
#240512 13:09:11 server id 1  end_log_pos 7667 CRC32 0x02d34168 	Write_rows: table id 119 flags: STMT_END_F

BINLOG '
d79AZh0BAAAAYgMAAP4bAACAAEpJTlNFUlQgSU5UTyB0ZXN0LmRhdGFfdHlwZXNfZGVtbyAoCiAg
IHZhcmNoYXJfY29sLCAKICAgY2hhcl9jb2wsIAogICBpbnRfY29sLCAKICAgc21hbGxpbnRfY29s
LCAKICAgdGlueWludF9jb2wsIAogICBiaWdpbnRfY29sLCAKICAgZmxvYXRfY29sLCAKICAgZG91
YmxlX2NvbCwgCiAgIGRlY2ltYWxfY29sLCAKICAgZGF0ZV9jb2wsIAogICB0aW1lX2NvbCwgCiAg
IGRhdGV0aW1lX2NvbCwgCiAgIHRpbWVzdGFtcF9jb2wsIAogICB5ZWFyX2NvbCwgCiAgIGJsb2Jf
Y29sLCAKICAgdGV4dF9jb2wsIAogICBlbnVtX2NvbCwgCiAgIHNldF9jb2wKICkgVkFMVUVTICgK
ICAgJ0V4YW1wbGUgdGV4dCcsICAgICAgICAgIAogICAnQUJDREUnLCAgICAgICAgICAgICAgICAg
CiAgIDEyMzQ1LCAgICAgICAgICAgICAgICAgICAKICAgMzI3NjcsICAgICAgICAgICAgICAgICAg
IAogICAxMjcsICAgICAgICAgICAgICAgICAgICAgCiAgIDkyMjMzNzIwMzY4NTQ3NzU4MDcsICAg
ICAKICAgMTIzNDUuNjc4LCAgICAgICAgICAgICAgIAogICAxMjM0NTY3OC45MTAxMSwgICAgICAg
ICAgCiAgIDEyMzQ1LjY3LCAgICAgICAgICAgICAgICAKICAgJzIwMjQtMDUtMTAnLCAgICAgICAg
ICAgIAogICAnMTU6MzA6MDAnLCAgICAgICAgICAgICAgCiAgICcyMDI0LTA1LTEwIDE1OjMwOjAw
JywgICAKICAgQ1VSUkVOVF9USU1FU1RBTVAsICAgICAgIAogICAyMDI0LCAgICAgICAgICAgICAg
ICAgICAgCiAgICdiaW5hcnkgZGF0YSBoZXJlJywgICAgICAKICAgJ0xvbmdlciBwaWVjZSBvZiB0
ZXh0JywgIAogICAndmFsMScsICAgICAgICAgICAgICAgICAgCiAgICdzZXQxICcgICAgICAgICAg
ICAgIAogKXp5Qng=
d79AZhMBAAAAWwEAAFkdAAAAAHcAAAAAAAEABHRlc3QAD2RhdGFfdHlwZXNfZGVtbwATAw/+AwIB
CAQF9goTEhEN/Pz+/hH8A/4oBAgKAgAAAAIC9wH4Af7/BwECAIACBfz/AAI/BL4CaWQLdmFyY2hh
cl9jb2wIY2hhcl9jb2wHaW50X2NvbAxzbWFsbGludF9jb2wLdGlueWludF9jb2wKYmlnaW50X2Nv
bAlmbG9hdF9jb2wKZG91YmxlX2NvbAtkZWNpbWFsX2NvbAhkYXRlX2NvbAh0aW1lX2NvbAxkYXRl
dGltZV9jb2wNdGltZXN0YW1wX2NvbAh5ZWFyX2NvbAhibG9iX2NvbAh0ZXh0X2NvbAhlbnVtX2Nv
bAdzZXRfY29sCgP8/wAFEAMEc2V0MQRzZXQyBHNldDMGEAMEdmFsMQR2YWwyBHZhbDMIAQAMA///
4CIZtek=
d79AZh4BAAAAmgAAAPMdAAAAAHcAAAAAAAEAAgAT////AAAAAwAAAAwARXhhbXBsZSB0ZXh0BUFC
Q0RFOTAAAP9/f/////////9/tuZARgKfH90pjGdBgAAwOUOq0A+A94CZs1T3gGZAv3d8EABiaW5h
cnkgZGF0YSBoZXJlFABMb25nZXIgcGllY2Ugb2YgdGV4dAEBaEHTAg==
'/*!*/;
### INSERT INTO `test`.`data_types_demo`
### SET
###   @1=3 /* INT meta=0 nullable=0 is_null=0 */
###   @2='Example text' /* VARSTRING(1020) meta=1020 nullable=1 is_null=0 */
###   @3='ABCDE' /* STRING(40) meta=65064 nullable=1 is_null=0 */
###   @4=12345 /* INT meta=0 nullable=1 is_null=0 */
###   @5=32767 /* SHORTINT meta=0 nullable=1 is_null=0 */
###   @6=127 /* TINYINT meta=0 nullable=1 is_null=0 */
###   @7=9223372036854775807 /* LONGINT meta=0 nullable=1 is_null=0 */
###   @8=12345.7              /* FLOAT meta=4 nullable=1 is_null=0 */
###   @9=12345678.910110000521 /* DOUBLE meta=8 nullable=1 is_null=0 */
###   @10=12345.67 /* DECIMAL(10,2) meta=2562 nullable=1 is_null=0 */
###   @11='2024:05:10' /* DATE meta=0 nullable=1 is_null=0 */
###   @12='15:30:00' /* TIME(0) meta=0 nullable=1 is_null=0 */
###   @13='2024-05-10 15:30:00' /* DATETIME(0) meta=0 nullable=1 is_null=0 */
###   @14=1715519351 /* TIMESTAMP(0) meta=0 nullable=1 is_null=0 */
###   @15=2024 /* YEAR meta=0 nullable=1 is_null=0 */
###   @16='binary data here' /* BLOB/TEXT meta=2 nullable=1 is_null=0 */
###   @17='Longer piece of text' /* BLOB/TEXT meta=2 nullable=1 is_null=0 */
###   @18=1 /* ENUM(1 byte) meta=63233 nullable=1 is_null=0 */
###   @19=b'00000001' /* SET(1 bytes) meta=63489 nullable=1 is_null=0 */
# at 7667
#240512 13:09:11 server id 1  end_log_pos 7698 CRC32 0x27c17152 	Xid = 188
COMMIT/*!*/;
# at 7698
#240512 13:09:18 server id 1  end_log_pos 7777 CRC32 0x2d2f4e73 	GTID	last_committed=8	sequence_number=9	rbr_only=yes	original_committed_timestamp=1715519358364029	immediate_commit_timestamp=1715519358364029	transaction_length=1556
/*!50718 SET TRANSACTION ISOLATION LEVEL READ COMMITTED*//*!*/;
# original_commit_timestamp=1715519358364029 (2024-05-12 13:09:18.364029 UTC)
# immediate_commit_timestamp=1715519358364029 (2024-05-12 13:09:18.364029 UTC)
/*!80001 SET @@session.original_commit_timestamp=1715519358364029*//*!*/;
/*!80014 SET @@session.original_server_version=80036*//*!*/;
/*!80014 SET @@session.immediate_server_version=80036*//*!*/;
SET @@SESSION.GTID_NEXT= '7f4be0b0-09b9-11ef-9bc6-0242ac130002:9'/*!*/;
# at 7777
#240512 13:09:18 server id 1  end_log_pos 7856 CRC32 0x2988b7ad 	Query	thread_id=58	exec_time=0	error_code=0
SET TIMESTAMP=1715519358/*!*/;
BEGIN
/*!*/;
# at 7856
#240512 13:09:18 server id 1  end_log_pos 8722 CRC32 0x9dc21283 	Rows_query
# INSERT INTO test.data_types_demo (    varchar_col,     char_col,     int_col,     smallint_col,     tinyint_col,     bigint_col,     float_col,     double_col,     decimal_col,     date_col,     time_col,     datetime_col,     timestamp_col,     year_col,     blob_col,     text_col,     enum_col,     set_col  ) VALUES (    'Example text',              'ABCDE',                     12345,                       32767,                       127,                         9223372036854775807,         12345.678,                   12345678.91011,              12345.67,                    '2024-05-10',                '15:30:00',                  '2024-05-10 15:30:00',       CURRENT_TIMESTAMP,           2024,                        'binary data here',          'Longer piece of text',      'val1',                      'set2 '                )
# at 8722
#240512 13:09:18 server id 1  end_log_pos 9069 CRC32 0xb95fb2a1 	Table_map: `test`.`data_types_demo` mapped to number 119
# has_generated_invisible_primary_key=0
# Columns(`id` INT NOT NULL,
#         `varchar_col` VARCHAR(255) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `char_col` CHAR(10) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `int_col` INT,
#         `smallint_col` SMALLINT,
#         `tinyint_col` TINYINT,
#         `bigint_col` BIGINT,
#         `float_col` FLOAT,
#         `double_col` DOUBLE,
#         `decimal_col` DECIMAL(10,2),
#         `date_col` DATE,
#         `time_col` TIME,
#         `datetime_col` DATETIME,
#         `timestamp_col` TIMESTAMP,
#         `year_col` YEAR,
#         `blob_col` BLOB,
#         `text_col` TEXT CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `enum_col` ENUM('val1', 'val2', 'val3') CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `set_col` SET('set1', 'set2', 'set3') CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci)
# Primary Key(id)
# at 9069
#240512 13:09:18 server id 1  end_log_pos 9223 CRC32 0xc8352f73 	Write_rows: table id 119 flags: STMT_END_F

BINLOG '
fr9AZh0BAAAAYgMAABIiAACAAEpJTlNFUlQgSU5UTyB0ZXN0LmRhdGFfdHlwZXNfZGVtbyAoICAg
IHZhcmNoYXJfY29sLCAgICAgY2hhcl9jb2wsICAgICBpbnRfY29sLCAgICAgc21hbGxpbnRfY29s
LCAgICAgdGlueWludF9jb2wsICAgICBiaWdpbnRfY29sLCAgICAgZmxvYXRfY29sLCAgICAgZG91
YmxlX2NvbCwgICAgIGRlY2ltYWxfY29sLCAgICAgZGF0ZV9jb2wsICAgICB0aW1lX2NvbCwgICAg
IGRhdGV0aW1lX2NvbCwgICAgIHRpbWVzdGFtcF9jb2wsICAgICB5ZWFyX2NvbCwgICAgIGJsb2Jf
Y29sLCAgICAgdGV4dF9jb2wsICAgICBlbnVtX2NvbCwgICAgIHNldF9jb2wgICkgVkFMVUVTICgg
ICAgJ0V4YW1wbGUgdGV4dCcsICAgICAgICAgICAgICAnQUJDREUnLCAgICAgICAgICAgICAgICAg
ICAgIDEyMzQ1LCAgICAgICAgICAgICAgICAgICAgICAgMzI3NjcsICAgICAgICAgICAgICAgICAg
ICAgICAxMjcsICAgICAgICAgICAgICAgICAgICAgICAgIDkyMjMzNzIwMzY4NTQ3NzU4MDcsICAg
ICAgICAgMTIzNDUuNjc4LCAgICAgICAgICAgICAgICAgICAxMjM0NTY3OC45MTAxMSwgICAgICAg
ICAgICAgIDEyMzQ1LjY3LCAgICAgICAgICAgICAgICAgICAgJzIwMjQtMDUtMTAnLCAgICAgICAg
ICAgICAgICAnMTU6MzA6MDAnLCAgICAgICAgICAgICAgICAgICcyMDI0LTA1LTEwIDE1OjMwOjAw
JywgICAgICAgQ1VSUkVOVF9USU1FU1RBTVAsICAgICAgICAgICAyMDI0LCAgICAgICAgICAgICAg
ICAgICAgICAgICdiaW5hcnkgZGF0YSBoZXJlJywgICAgICAgICAgJ0xvbmdlciBwaWVjZSBvZiB0
ZXh0JywgICAgICAndmFsMScsICAgICAgICAgICAgICAgICAgICAgICdzZXQyICcgICAgICAgICAg
ICAgICAgKYMSwp0=
fr9AZhMBAAAAWwEAAG0jAAAAAHcAAAAAAAEABHRlc3QAD2RhdGFfdHlwZXNfZGVtbwATAw/+AwIB
CAQF9goTEhEN/Pz+/hH8A/4oBAgKAgAAAAIC9wH4Af7/BwECAIACBfz/AAI/BL4CaWQLdmFyY2hh
cl9jb2wIY2hhcl9jb2wHaW50X2NvbAxzbWFsbGludF9jb2wLdGlueWludF9jb2wKYmlnaW50X2Nv
bAlmbG9hdF9jb2wKZG91YmxlX2NvbAtkZWNpbWFsX2NvbAhkYXRlX2NvbAh0aW1lX2NvbAxkYXRl
dGltZV9jb2wNdGltZXN0YW1wX2NvbAh5ZWFyX2NvbAhibG9iX2NvbAh0ZXh0X2NvbAhlbnVtX2Nv
bAdzZXRfY29sCgP8/wAFEAMEc2V0MQRzZXQyBHNldDMGEAMEdmFsMQR2YWwyBHZhbDMIAQAMA///
4KGyX7k=
fr9AZh4BAAAAmgAAAAckAAAAAHcAAAAAAAEAAgAT////AAAABAAAAAwARXhhbXBsZSB0ZXh0BUFC
Q0RFOTAAAP9/f/////////9/tuZARgKfH90pjGdBgAAwOUOq0A+A94CZs1T3gGZAv358EABiaW5h
cnkgZGF0YSBoZXJlFABMb25nZXIgcGllY2Ugb2YgdGV4dAECcy81yA==
'/*!*/;
### INSERT INTO `test`.`data_types_demo`
### SET
###   @1=4 /* INT meta=0 nullable=0 is_null=0 */
###   @2='Example text' /* VARSTRING(1020) meta=1020 nullable=1 is_null=0 */
###   @3='ABCDE' /* STRING(40) meta=65064 nullable=1 is_null=0 */
###   @4=12345 /* INT meta=0 nullable=1 is_null=0 */
###   @5=32767 /* SHORTINT meta=0 nullable=1 is_null=0 */
###   @6=127 /* TINYINT meta=0 nullable=1 is_null=0 */
###   @7=9223372036854775807 /* LONGINT meta=0 nullable=1 is_null=0 */
###   @8=12345.7              /* FLOAT meta=4 nullable=1 is_null=0 */
###   @9=12345678.910110000521 /* DOUBLE meta=8 nullable=1 is_null=0 */
###   @10=12345.67 /* DECIMAL(10,2) meta=2562 nullable=1 is_null=0 */
###   @11='2024:05:10' /* DATE meta=0 nullable=1 is_null=0 */
###   @12='15:30:00' /* TIME(0) meta=0 nullable=1 is_null=0 */
###   @13='2024-05-10 15:30:00' /* DATETIME(0) meta=0 nullable=1 is_null=0 */
###   @14=1715519358 /* TIMESTAMP(0) meta=0 nullable=1 is_null=0 */
###   @15=2024 /* YEAR meta=0 nullable=1 is_null=0 */
###   @16='binary data here' /* BLOB/TEXT meta=2 nullable=1 is_null=0 */
###   @17='Longer piece of text' /* BLOB/TEXT meta=2 nullable=1 is_null=0 */
###   @18=1 /* ENUM(1 byte) meta=63233 nullable=1 is_null=0 */
###   @19=b'00000010' /* SET(1 bytes) meta=63489 nullable=1 is_null=0 */
# at 9223
#240512 13:09:18 server id 1  end_log_pos 9254 CRC32 0xc839fca1 	Xid = 189
COMMIT/*!*/;
# at 9254
#240512 13:09:26 server id 1  end_log_pos 9333 CRC32 0x56787203 	GTID	last_committed=9	sequence_number=10	rbr_only=yes	original_committed_timestamp=1715519366279552	immediate_commit_timestamp=1715519366279552	transaction_length=1565
/*!50718 SET TRANSACTION ISOLATION LEVEL READ COMMITTED*//*!*/;
# original_commit_timestamp=1715519366279552 (2024-05-12 13:09:26.279552 UTC)
# immediate_commit_timestamp=1715519366279552 (2024-05-12 13:09:26.279552 UTC)
/*!80001 SET @@session.original_commit_timestamp=1715519366279552*//*!*/;
/*!80014 SET @@session.original_server_version=80036*//*!*/;
/*!80014 SET @@session.immediate_server_version=80036*//*!*/;
SET @@SESSION.GTID_NEXT= '7f4be0b0-09b9-11ef-9bc6-0242ac130002:10'/*!*/;
# at 9333
#240512 13:09:26 server id 1  end_log_pos 9412 CRC32 0x8137da75 	Query	thread_id=58	exec_time=0	error_code=0
SET TIMESTAMP=1715519366/*!*/;
BEGIN
/*!*/;
# at 9412
#240512 13:09:26 server id 1  end_log_pos 10287 CRC32 0x46d3bca2 	Rows_query
# INSERT INTO test.data_types_demo (    varchar_col,     char_col,     int_col,     smallint_col,     tinyint_col,     bigint_col,     float_col,     double_col,     decimal_col,     date_col,     time_col,     datetime_col,     timestamp_col,     year_col,     blob_col,     text_col,     enum_col,     set_col  ) VALUES (    'Example text',              'ABCDE',                     12345,                       32767,                       127,                         9223372036854775807,         12345.678,                   12345678.91011,              12345.67,                    '2024-05-10',                '15:30:00',                  '2024-05-10 15:30:00',       CURRENT_TIMESTAMP,           2024,                        'binary data here',          'Longer piece of text',      'val1',                      'set2,set3,set1'                )
# at 10287
#240512 13:09:26 server id 1  end_log_pos 10634 CRC32 0x9bb97082 	Table_map: `test`.`data_types_demo` mapped to number 119
# has_generated_invisible_primary_key=0
# Columns(`id` INT NOT NULL,
#         `varchar_col` VARCHAR(255) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `char_col` CHAR(10) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `int_col` INT,
#         `smallint_col` SMALLINT,
#         `tinyint_col` TINYINT,
#         `bigint_col` BIGINT,
#         `float_col` FLOAT,
#         `double_col` DOUBLE,
#         `decimal_col` DECIMAL(10,2),
#         `date_col` DATE,
#         `time_col` TIME,
#         `datetime_col` DATETIME,
#         `timestamp_col` TIMESTAMP,
#         `year_col` YEAR,
#         `blob_col` BLOB,
#         `text_col` TEXT CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `enum_col` ENUM('val1', 'val2', 'val3') CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci,
#         `set_col` SET('set1', 'set2', 'set3') CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci)
# Primary Key(id)
# at 10634
#240512 13:09:26 server id 1  end_log_pos 10788 CRC32 0x3fbda99e 	Write_rows: table id 119 flags: STMT_END_F

BINLOG '
hr9AZh0BAAAAawMAAC8oAACAAFNJTlNFUlQgSU5UTyB0ZXN0LmRhdGFfdHlwZXNfZGVtbyAoICAg
IHZhcmNoYXJfY29sLCAgICAgY2hhcl9jb2wsICAgICBpbnRfY29sLCAgICAgc21hbGxpbnRfY29s
LCAgICAgdGlueWludF9jb2wsICAgICBiaWdpbnRfY29sLCAgICAgZmxvYXRfY29sLCAgICAgZG91
YmxlX2NvbCwgICAgIGRlY2ltYWxfY29sLCAgICAgZGF0ZV9jb2wsICAgICB0aW1lX2NvbCwgICAg
IGRhdGV0aW1lX2NvbCwgICAgIHRpbWVzdGFtcF9jb2wsICAgICB5ZWFyX2NvbCwgICAgIGJsb2Jf
Y29sLCAgICAgdGV4dF9jb2wsICAgICBlbnVtX2NvbCwgICAgIHNldF9jb2wgICkgVkFMVUVTICgg
ICAgJ0V4YW1wbGUgdGV4dCcsICAgICAgICAgICAgICAnQUJDREUnLCAgICAgICAgICAgICAgICAg
ICAgIDEyMzQ1LCAgICAgICAgICAgICAgICAgICAgICAgMzI3NjcsICAgICAgICAgICAgICAgICAg
ICAgICAxMjcsICAgICAgICAgICAgICAgICAgICAgICAgIDkyMjMzNzIwMzY4NTQ3NzU4MDcsICAg
ICAgICAgMTIzNDUuNjc4LCAgICAgICAgICAgICAgICAgICAxMjM0NTY3OC45MTAxMSwgICAgICAg
ICAgICAgIDEyMzQ1LjY3LCAgICAgICAgICAgICAgICAgICAgJzIwMjQtMDUtMTAnLCAgICAgICAg
ICAgICAgICAnMTU6MzA6MDAnLCAgICAgICAgICAgICAgICAgICcyMDI0LTA1LTEwIDE1OjMwOjAw
JywgICAgICAgQ1VSUkVOVF9USU1FU1RBTVAsICAgICAgICAgICAyMDI0LCAgICAgICAgICAgICAg
ICAgICAgICAgICdiaW5hcnkgZGF0YSBoZXJlJywgICAgICAgICAgJ0xvbmdlciBwaWVjZSBvZiB0
ZXh0JywgICAgICAndmFsMScsICAgICAgICAgICAgICAgICAgICAgICdzZXQyLHNldDMsc2V0MScg
ICAgICAgICAgICAgICAgKaK800Y=
hr9AZhMBAAAAWwEAAIopAAAAAHcAAAAAAAEABHRlc3QAD2RhdGFfdHlwZXNfZGVtbwATAw/+AwIB
CAQF9goTEhEN/Pz+/hH8A/4oBAgKAgAAAAIC9wH4Af7/BwECAIACBfz/AAI/BL4CaWQLdmFyY2hh
cl9jb2wIY2hhcl9jb2wHaW50X2NvbAxzbWFsbGludF9jb2wLdGlueWludF9jb2wKYmlnaW50X2Nv
bAlmbG9hdF9jb2wKZG91YmxlX2NvbAtkZWNpbWFsX2NvbAhkYXRlX2NvbAh0aW1lX2NvbAxkYXRl
dGltZV9jb2wNdGltZXN0YW1wX2NvbAh5ZWFyX2NvbAhibG9iX2NvbAh0ZXh0X2NvbAhlbnVtX2Nv
bAdzZXRfY29sCgP8/wAFEAMEc2V0MQRzZXQyBHNldDMGEAMEdmFsMQR2YWwyBHZhbDMIAQAMA///
4IJwuZs=
hr9AZh4BAAAAmgAAACQqAAAAAHcAAAAAAAEAAgAT////AAAABQAAAAwARXhhbXBsZSB0ZXh0BUFC
Q0RFOTAAAP9/f/////////9/tuZARgKfH90pjGdBgAAwOUOq0A+A94CZs1T3gGZAv4Z8EABiaW5h
cnkgZGF0YSBoZXJlFABMb25nZXIgcGllY2Ugb2YgdGV4dAEHnqm9Pw==
'/*!*/;
### INSERT INTO `test`.`data_types_demo`
### SET
###   @1=5 /* INT meta=0 nullable=0 is_null=0 */
###   @2='Example text' /* VARSTRING(1020) meta=1020 nullable=1 is_null=0 */
###   @3='ABCDE' /* STRING(40) meta=65064 nullable=1 is_null=0 */
###   @4=12345 /* INT meta=0 nullable=1 is_null=0 */
###   @5=32767 /* SHORTINT meta=0 nullable=1 is_null=0 */
###   @6=127 /* TINYINT meta=0 nullable=1 is_null=0 */
###   @7=9223372036854775807 /* LONGINT meta=0 nullable=1 is_null=0 */
###   @8=12345.7              /* FLOAT meta=4 nullable=1 is_null=0 */
###   @9=12345678.910110000521 /* DOUBLE meta=8 nullable=1 is_null=0 */
###   @10=12345.67 /* DECIMAL(10,2) meta=2562 nullable=1 is_null=0 */
###   @11='2024:05:10' /* DATE meta=0 nullable=1 is_null=0 */
###   @12='15:30:00' /* TIME(0) meta=0 nullable=1 is_null=0 */
###   @13='2024-05-10 15:30:00' /* DATETIME(0) meta=0 nullable=1 is_null=0 */
###   @14=1715519366 /* TIMESTAMP(0) meta=0 nullable=1 is_null=0 */
###   @15=2024 /* YEAR meta=0 nullable=1 is_null=0 */
###   @16='binary data here' /* BLOB/TEXT meta=2 nullable=1 is_null=0 */
###   @17='Longer piece of text' /* BLOB/TEXT meta=2 nullable=1 is_null=0 */
###   @18=1 /* ENUM(1 byte) meta=63233 nullable=1 is_null=0 */
###   @19=b'00000111' /* SET(1 bytes) meta=63489 nullable=1 is_null=0 */
# at 10788
#240512 13:09:26 server id 1  end_log_pos 10819 CRC32 0xa8a4c6bd 	Xid = 190
COMMIT/*!*/;
SET @@SESSION.GTID_NEXT= 'AUTOMATIC' /* added by mysqlbinlog */ /*!*/;
DELIMITER ;
# End of log file
/*!50003 SET COMPLETION_TYPE=@OLD_COMPLETION_TYPE*/;
/*!50530 SET @@SESSION.PSEUDO_SLAVE_MODE=0*/;
/*!50700 SET @@SESSION.RBR_EXEC_MODE=STRICT*/;
