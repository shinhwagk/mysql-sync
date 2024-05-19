from pymysqlreplication import BinLogStreamReader
from pymysqlreplication.event import GtidEvent
from pymysqlreplication.row_event import (
    TableMapEvent,
    DeleteRowsEvent,
    UpdateRowsEvent,
    WriteRowsEvent,
)
from typing import cast

from pymysqlreplication.column import Column

mysql_settings = {"host": "db1", "port": 3306, "user": "root", "passwd": "root_password"}

stream = BinLogStreamReader(connection_settings=mysql_settings, server_id=100)


def handle_table_map_event(event: TableMapEvent):
    pass
    # Logic for TableMapEvent


def handle_update_rows_event(event: UpdateRowsEvent):
    pass
    # Logic for UpdateRowsEvent


def handle_write_rows_event(event: WriteRowsEvent):
    pass
    # Logic for WriteRowsEvent


handlers = {TableMapEvent: handle_table_map_event, UpdateRowsEvent: handle_update_rows_event, WriteRowsEvent: handle_write_rows_event}

for binlogevent in stream:
    print(binlogevent)
    print(binlogevent.rows)
    print("+=========")
    handler = handlers.get(type(binlogevent))
    if handler:
        handler(binlogevent)
