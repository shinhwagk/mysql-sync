### Query & Xid

```sql
mysql-bin.000003	2296	Query	1	2436	create table test.tab1(a int primary key, b varchar(10)) /* xid=29 */
--
# at 2296
#240427  6:03:40 server id 1  end_log_pos 2436 CRC32 0x6c4d05c8 	Query	thread_id=20	exec_time=0	error_code=0	Xid = 29
SET TIMESTAMP=1714197820/*!*/;
/*!80013 SET @@session.sql_require_primary_key=0*//*!*/;
create table test.tab1(a int primary key, b varchar(10))
/*!*/;


mysql-bin.000006	776	Query	1	898	use `test`; alter table tab1 add c varchar(10) /* xid=98 */
--
 # at 776
#240428  8:07:06 server id 1  end_log_pos 898 CRC32 0xdba96913  Query   thread_id=27    exec_time=0     error_code=0    Xid = 98
use `test`/*!*/;
SET TIMESTAMP=1714291626/*!*/;
/*!80013 SET @@session.sql_require_primary_key=0*//*!*/;
alter table tab1 add c varchar(10)
/*!*/;
```

### Gtid
```sql
mysql-bin.000003	2436	Gtid	1	2515	SET @@SESSION.GTID_NEXT= 'cda8356d-0452-11ef-946d-0242ac120002:15'
---
# at 2436
#240427  6:03:45 server id 1  end_log_pos 2515 CRC32 0x0145d94f 	GTID	last_committed=9	sequence_number=10	rbr_only=yes	original_committed_timestamp=1714197825620789	immediate_commit_timestamp=1714197825620789	transaction_length=281
/*!50718 SET TRANSACTION ISOLATION LEVEL READ COMMITTED*//*!*/;
# original_commit_timestamp=1714197825620789 (2024-04-27 06:03:45.620789 UTC)
# immediate_commit_timestamp=1714197825620789 (2024-04-27 06:03:45.620789 UTC)
/*!80001 SET @@session.original_commit_timestamp=1714197825620789*//*!*/;
/*!80014 SET @@session.original_server_version=80036*//*!*/;
/*!80014 SET @@session.immediate_server_version=80036*//*!*/;
SET @@SESSION.GTID_NEXT= 'cda8356d-0452-11ef-946d-0242ac120002:15'/*!*/;
```

### Query | Table_map | Write_rows | Xid
```sql
mysql-bin.000003	2515	Query	1	2586	BEGIN
mysql-bin.000003	2586	Table_map	1	2644	table_id: 88 (test.tab1)
mysql-bin.000003	2644	Write_rows	1	2686	table_id: 88 flags: STMT_END_F
mysql-bin.000003	2686	Xid	1	2717	COMMIT /* xid=30 */
---
# at 2515
#240427  6:03:45 server id 1  end_log_pos 2586 CRC32 0x831776ed 	Query	thread_id=20	exec_time=0	error_code=0
SET TIMESTAMP=1714197825/*!*/;
BEGIN
/*!*/;
# at 2586
#240427  6:03:45 server id 1  end_log_pos 2644 CRC32 0x4f251d70 	Table_map: `test`.`tab1` mapped to number 88
# has_generated_invisible_primary_key=0
# at 2644
#240427  6:03:45 server id 1  end_log_pos 2686 CRC32 0x5eb28bd5 	Write_rows: table id 88 flags: STMT_END_F

BINLOG '
QZUsZhMBAAAAOgAAAFQKAAAAAFgAAAAAAAEABHRlc3QABHRhYjEAAgMPAigAAgEBAAID/P8AcB0l
Tw==
QZUsZh4BAAAAKgAAAH4KAAAAAFgAAAAAAAEAAgAC/wABAAAAAWLVi7Je
'/*!*/;
### INSERT INTO `test`.`tab1`
### SET
###   @1=1 /* INT meta=0 nullable=0 is_null=0 */
###   @2='b' /* VARSTRING(40) meta=40 nullable=1 is_null=0 */
# at 2686
#240427  6:03:45 server id 1  end_log_pos 2717 CRC32 0x23377cc8 	Xid = 30
COMMIT/*!*/;
```

### Previous_gtids
```sql
mysql-bin.000006	126	Previous_gtids	1	197	cda8356d-0452-11ef-946d-0242ac120002:1-16
---
# at 126
#240428  7:39:24 server id 1  end_log_pos 197 CRC32 0x270ee81b 	Previous-GTIDs
# cda8356d-0452-11ef-946d-0242ac120002:1-16
```

### Start
```sql
mysql-bin.000006	4	Format_desc	1	126	Server ver: 8.0.36, Binlog ver: 4
---
# at 4
#240428  7:39:24 server id 1  end_log_pos 126 CRC32 0xc3857e5f 	Start: binlog v 4, server v 8.0.36 created 240428  7:39:24
BINLOG '
LP0tZg8BAAAAegAAAH4AAAAAAAQAOC4wLjM2AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAEwANAAgAAAAABAAEAAAAYgAEGggAAAAICAgCAAAACgoKKioAEjQA
CigAAV9+hcM=
'/*!*/;
```

### Rotate
```sql
mysql-bin.000005	197	Rotate	1	244	mysql-bin.000006;pos=4
---
# at 197
#240428  7:39:03 server id 1  end_log_pos 244 CRC32 0xc89cb9c8 	Rotate to mysql-bin.000005  pos: 4
```

### Stop
```sql
mysql-bin.000002	3040324	Stop	1	3040347	
-- 
# at 3040324
#240427  4:59:08 server id 1  end_log_pos 3040347 CRC32 0x5be9e6e4 	Stop

```