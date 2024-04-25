# The proper term is pseudo_replica_mode, but we use this compatibility alias
# to make the statement usable on server versions 8.0.24 and older.
/*!50530 SET @@SESSION.PSEUDO_SLAVE_MODE=1*/;
/*!50003 SET @OLD_COMPLETION_TYPE=@@COMPLETION_TYPE,COMPLETION_TYPE=0*/;
/*!50700 SET @@SESSION.RBR_EXEC_MODE=IDEMPOTENT*/;

DELIMITER /*!*/;
# at 4
#700101  0:00:00 server id 1  end_log_pos 0 	Rotate to mysql-bin.000006  pos: 4
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
