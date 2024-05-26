from pymysqlreplication.event import RowsQueryLogEvent

source_settings = {
    "connection_settings": {
        "host": "db1",
        "port": 3306,
        "user": "root",
        "password": "root_password",
        "charset": "utf8mb4",
    },
    "server_id": 9999,
    "ignored_events": [RowsQueryLogEvent],
    "blocking": True,
    "report_slave": "myqslsync",
    "slave_heartbeat": 10,
    "auto_position": "a8d55263-1a60-11ef-b60e-0242ac130006:1-179",
}
target_settings = {
    "connection_settings": {
        "host": "db2",
        "port": 3306,
        "user": "root",
        "password": "root_password",
        "charset": "utf8mb4",
        # "compress": True,
    }
}

sync_settings = {"merge_trx": True}
