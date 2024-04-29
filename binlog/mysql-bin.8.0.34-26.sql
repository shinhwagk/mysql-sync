# The proper term is pseudo_replica_mode, but we use this compatibility alias
# to make the statement usable on server versions 8.0.24 and older.
/*!50530 SET @@SESSION.PSEUDO_SLAVE_MODE=1*/;
/*!50003 SET @OLD_COMPLETION_TYPE=@@COMPLETION_TYPE,COMPLETION_TYPE=0*/;
/*!50700 SET @@SESSION.RBR_EXEC_MODE=IDEMPOTENT*/;

DELIMITER /*!*/;
# at 4
#691231 16:00:00 server id 161933306  end_log_pos 0 	Rotate to mysql-bin.000029  pos: 4
# at 4
[root@cd-uat-litb-db-6 litbzeus]# cat mysql-bin.log | head -100
# The proper term is pseudo_replica_mode, but we use this compatibility alias
# to make the statement usable on server versions 8.0.24 and older.
/*!50530 SET @@SESSION.PSEUDO_SLAVE_MODE=1*/;
/*!50003 SET @OLD_COMPLETION_TYPE=@@COMPLETION_TYPE,COMPLETION_TYPE=0*/;
/*!50700 SET @@SESSION.RBR_EXEC_MODE=IDEMPOTENT*/;

DELIMITER /*!*/;
# at 4
#691231 16:00:00 server id 161933306  end_log_pos 0 	Rotate to mysql-bin.000029  pos: 4
# at 4
#240413  5:14:12 server id 161933306  end_log_pos 126 CRC32 0x7e7a0b70 	Start: binlog v 4, server v 8.0.34-26 created 240413  5:14:12
BINLOG '
FHcaZg/656YJegAAAH4AAAAAAAQAOC4wLjM0LTI2AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAEwANAAgAAAAABAAEAAAAYgAEGggAAAAICAgCAAAACgoKKioAEjQA
CigAAXALen4=
'/*!*/;
# at 126
#240413  5:14:12 server id 161933306  end_log_pos 285 CRC32 0x5824c380 	Previous-GTIDs
# 93e75241-927c-11ee-ac1a-2cea7fe83470:386552-522009:522011-522022:522024-522038:522040-1921532,
# b3eaf2df-9334-11ee-970f-2cea7fe83470:12054-1223173
# at 285
#240413  5:14:13 server id 161183306  end_log_pos 371 CRC32 0x0ada3fca 	GTID	last_committed=0	sequence_number=1	rbr_only=yes	original_committed_timestamp=1713010453063823	immediate_commit_timestamp=1713010453106729	transaction_length=1908
/*!50718 SET TRANSACTION ISOLATION LEVEL READ COMMITTED*//*!*/;
# original_commit_timestamp=1713010453063823 (2024-04-13 05:14:13.063823 PDT)
# immediate_commit_timestamp=1713010453106729 (2024-04-13 05:14:13.106729 PDT)
/*!80001 SET @@session.original_commit_timestamp=1713010453063823*//*!*/;
/*!80014 SET @@session.original_server_version=80034*//*!*/;
/*!80014 SET @@session.immediate_server_version=80034*//*!*/;
SET @@SESSION.GTID_NEXT= '93e75241-927c-11ee-ac1a-2cea7fe83470:1921533'/*!*/;
# at 371
#240413  5:14:13 server id 161183306  end_log_pos 460 CRC32 0xbd59fa5a 	Query	thread_id=202	exec_time=0	error_code=0
SET TIMESTAMP=1713010453/*!*/;
SET @@session.pseudo_thread_id=202/*!*/;
SET @@session.foreign_key_checks=1, @@session.sql_auto_is_null=0, @@session.unique_checks=1, @@session.autocommit=1/*!*/;
SET @@session.sql_mode=1074266112/*!*/;
SET @@session.auto_increment_increment=1, @@session.auto_increment_offset=1/*!*/;
/*!\C utf8mb4 *//*!*/;
SET @@session.character_set_client=45,@@session.collation_connection=45,@@session.collation_server=45/*!*/;
SET @@session.lc_time_names=0/*!*/;
SET @@session.collation_database=DEFAULT/*!*/;
/*!80011 SET @@session.default_collation_for_utf8mb4=45*//*!*/;
BEGIN
/*!*/;
# at 460
#240413  5:14:13 server id 161183306  end_log_pos 1518 CRC32 0x9f34ebaa 	Rows_query
# insert into `merchant_center_vela_v1`.`v3_express_recommend`(`dest_country_id` , `warehouse_id` , `weight` , `shipping_module_code` , `express_id` , `price` , `status` , `last_modified` , `prom_expense` , `allowance` , `id`) values (22 , 67 , 1.85 , 'Expedited' , 2 , 403.98 , 1 , '2024-04-13 20:14:11' , 0.0 , 0.0 , 106052858) ,(22 , 67 , 1.7 , 'Expedited' , 2 , 403.98 , 1 , '2024-04-13 20:14:11' , 0.0 , 0.0 , 106052855) ,(22 , 67 , 1.8 , 'Expedited' , 2 , 403.98 , 1 , '2024-04-13 20:14:11' , 0.0 , 0.0 , 106052857) ,(22 , 67 , 1.9 , 'Expedited' , 2 , 403.98 , 1 , '2024-04-13 20:14:11' , 0.0 , 0.0 , 106052859)  on duplicate key update `dest_country_id`=values(`dest_country_id`) , `warehouse_id`=values(`warehouse_id`) , `weight`=values(`weight`) , `shipping_module_code`=values(`shipping_module_code`) , `express_id`=values(`express_id`) , `price`=values(`price`) , `status`=values(`status`) , `last_modified`=values(`last_modified`) , `prom_expense`=values(`prom_expense`) , `allowance`=values(`allowance`) , `id`=values(`id`)
# at 1518
#240413  5:14:13 server id 161183306  end_log_pos 1628 CRC32 0x33192119 	Table_map: `merchant_center_vela_v1`.`v3_express_recommend` mapped to number 394
# has_generated_invisible_primary_key=0
# Columns(INT NOT NULL,
#         INT NOT NULL,
#         INT NOT NULL,
#         DOUBLE NOT NULL,
#         VARCHAR(32) NOT NULL CHARSET utf8mb4 COLLATE utf8mb4_general_ci,
#         INT NOT NULL,
#         DECIMAL(14,2) NOT NULL,
#         TINYINT NOT NULL,
#         TIMESTAMP NOT NULL,
#         DECIMAL(14,2) NOT NULL,
#         DECIMAL(14,2) NOT NULL)
# at 1628
#240413  5:14:13 server id 161183306  end_log_pos 2162 CRC32 0x6b65e044 	Update_rows: table id 394 flags: STMT_END_F

BINLOG '
FXcaZh1KdpsJIgQAAO4FAACAAAppbnNlcnQgaW50byBgbWVyY2hhbnRfY2VudGVyX3ZlbGFfdjFg
LmB2M19leHByZXNzX3JlY29tbWVuZGAoYGRlc3RfY291bnRyeV9pZGAgLCBgd2FyZWhvdXNlX2lk
YCAsIGB3ZWlnaHRgICwgYHNoaXBwaW5nX21vZHVsZV9jb2RlYCAsIGBleHByZXNzX2lkYCAsIGBw
cmljZWAgLCBgc3RhdHVzYCAsIGBsYXN0X21vZGlmaWVkYCAsIGBwcm9tX2V4cGVuc2VgICwgYGFs
bG93YW5jZWAgLCBgaWRgKSB2YWx1ZXMgKDIyICwgNjcgLCAxLjg1ICwgJ0V4cGVkaXRlZCcgLCAy
ICwgNDAzLjk4ICwgMSAsICcyMDI0LTA0LTEzIDIwOjE0OjExJyAsIDAuMCAsIDAuMCAsIDEwNjA1
Mjg1OCkgLCgyMiAsIDY3ICwgMS43ICwgJ0V4cGVkaXRlZCcgLCAyICwgNDAzLjk4ICwgMSAsICcy
MDI0LTA0LTEzIDIwOjE0OjExJyAsIDAuMCAsIDAuMCAsIDEwNjA1Mjg1NSkgLCgyMiAsIDY3ICwg
MS44ICwgJ0V4cGVkaXRlZCcgLCAyICwgNDAzLjk4ICwgMSAsICcyMDI0LTA0LTEzIDIwOjE0OjEx
JyAsIDAuMCAsIDAuMCAsIDEwNjA1Mjg1NykgLCgyMiAsIDY3ICwgMS45ICwgJ0V4cGVkaXRlZCcg
LCAyICwgNDAzLjk4ICwgMSAsICcyMDI0LTA0LTEzIDIwOjE0OjExJyAsIDAuMCAsIDAuMCAsIDEw
NjA1Mjg1OSkgIG9uIGR1cGxpY2F0ZSBrZXkgdXBkYXRlIGBkZXN0X2NvdW50cnlfaWRgPXZhbHVl
cyhgZGVzdF9jb3VudHJ5X2lkYCkgLCBgd2FyZWhvdXNlX2lkYD12YWx1ZXMoYHdhcmVob3VzZV9p
ZGApICwgYHdlaWdodGA9dmFsdWVzKGB3ZWlnaHRgKSAsIGBzaGlwcGluZ19tb2R1bGVfY29kZWA9
dmFsdWVzKGBzaGlwcGluZ19tb2R1bGVfY29kZWApICwgYGV4cHJlc3NfaWRgPXZhbHVlcyhgZXhw
cmVzc19pZGApICwgYHByaWNlYD12YWx1ZXMoYHByaWNlYCkgLCBgc3RhdHVzYD12YWx1ZXMoYHN0
YXR1c2ApICwgYGxhc3RfbW9kaWZpZWRgPXZhbHVlcyhgbGFzdF9tb2RpZmllZGApICwgYHByb21f
ZXhwZW5zZWA9dmFsdWVzKGBwcm9tX2V4cGVuc2VgKSAsIGBhbGxvd2FuY2VgPXZhbHVlcyhgYWxs
b3dhbmNlYCkgLCBgaWRgPXZhbHVlcyhgaWRgKarrNJ8=
FXcaZhNKdpsJbgAAAFwGAAAAAIoBAAAAAAEAF21lcmNoYW50X2NlbnRlcl92ZWxhX3YxABR2M19l
eHByZXNzX3JlY29tbWVuZAALAwMDBQ8D9gER9vYKCIAADgIADgIOAgAAAQIAAAIBLRkhGTM=
FXcaZh9KdpsJFgIAAHIIAAAAAIoBAAAAAAEAAgAL/////wAA+jxSBhYAAABDAAAAmpmZmZmZ/T8J
RXhwZWRpdGVkAgAAAIAAAAABk2IBZhp0pIAAAAAAAACAAAAAAAAAAAD6PFIGFgAAAEMAAACamZmZ
mZn9PwlFeHBlZGl0ZWQCAAAAgAAAAAGTYgFmGncTgAAAAAAAAIAAAAAAAAAAAPc8UgYWAAAAQwAA
ADMzMzMzM/s/CUV4cGVkaXRlZAIAAACAAAAAAZNiAWYadKSAAAAAAAAAgAAAAAAAAAAA9zxSBhYA
AABDAAAAMzMzMzMz+z8JRXhwZWRpdGVkAgAAAIAAAAABk2IBZhp3E4AAAAAAAACAAAAAAAAAAAD5
PFIGFgAAAEMAAADNzMzMzMz8PwlFeHBlZGl0ZWQCAAAAgAAAAAGTYgFmGnSkgAAAAAAAAIAAAAAA
AAAAAPk8UgYWAAAAQwAAAM3MzMzMzPw/CUV4cGVkaXRlZAIAAACAAAAAAZNiAWYadxOAAAAAAAAA
gAAAAAAAAAAA+zxSBhYAAABDAAAAZmZmZmZm/j8JRXhwZWRpdGVkAgAAAIAAAAABk2IBZhp0pIAA
AAAAAACAAAAAAAAAAAD7PFIGFgAAAEMAAABmZmZmZmb+PwlFeHBlZGl0ZWQCAAAAgAAAAAGTYgFm
GncTgAAAAAAAAIAAAAAAAABE4GVr
'/*!*/;
### UPDATE `merchant_center_vela_v1`.`v3_express_recommend`
### WHERE
###   @1=106052858 /* INT meta=0 nullable=0 is_null=0 */
###   @2=22 /* INT meta=0 nullable=0 is_null=0 */