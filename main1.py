import argparse
import re
from collections import deque
from collections.abc import Iterator
from dataclasses import dataclass
from typing import Optional, Pattern, cast

import mysql.connector
from mysql.connector import MySQLConnection
from mysql.connector.cursor import MySQLCursor
from pymysqlreplication import BinLogStreamReader
from pymysqlreplication.column import Column
from pymysqlreplication.event import (
    BinLogEvent,
    GtidEvent,
    HeartbeatLogEvent,
    QueryEvent,
    RotateEvent,
    RowsQueryLogEvent,
    XidEvent,
)
from pymysqlreplication.row_event import (
    DeleteRowsEvent,
    TableMapEvent,
    UpdateRowsEvent,
    WriteRowsEvent,
)


def parse_connection_string(conn_str: str) -> dict:
    result = {}

    user_pass_part, host_part = conn_str.split("@")

    user, passwd = user_pass_part.split("/")
    result["user"] = user
    result["password"] = passwd

    if "?" in host_part:
        host_port, params = host_part.split("?")

        params_dict = dict(param.split("=") for param in params.split("&"))
        result.update(params_dict)
    else:
        host_port = host_part

    host, port = host_port.split(":")
    result["host"] = host
    result["port"] = int(port)

    result.setdefault("charset", "utf8mb4")

    return result


def generate_question_marks(n):
    return ",".join(f'{"%s"}' for _ in range(n))


def deleteRowsEvent2Sql(event: DeleteRowsEvent) -> str:
    pass


def updateRowsEvent2Sql(event: UpdateRowsEvent) -> str:
    pass


def updateRowsEvent2Replace(event: UpdateRowsEvent) -> str:
    pass


def writeRowsEvent2Insert(event: WriteRowsEvent) -> str:
    pass


def writeRowsEvent2Replace(event: WriteRowsEvent) -> str:
    pass


import enum


class DDLType(enum.Enum):
    CREATEDATABASE = enum.auto()
    DROPDATABASE = enum.auto()
    ALTERDATABASE = enum.auto()
    CREATETABLE = enum.auto()
    ALTERTABLE = enum.auto()
    DROPTABLE = enum.auto()
    RENAMETABLE = enum.auto()
    TRUNCATETABLE = enum.auto()
    CREATEINDEX = enum.auto()
    ALTERINDEX = enum.auto()
    DROPINDEX = enum.auto()


@dataclass
class OperationDDL:
    schema: str | None
    sql_text: str


@dataclass
class OperationDML:
    sql_text: str
    params: tuple


@dataclass
class OperationHeartbeat:
    pass


@dataclass
class OperationCommit:
    pass


@dataclass
class OperationBegin:
    pass


@dataclass
class OperationGtid:
    gtid: str
    last_committed: int


class OperationType(enum.Enum):
    DDL = enum.auto()
    BEGIN = enum.auto()
    COMMIT = enum.auto()
    DML = enum.auto()
    GTID = enum.auto()


def extract_schema(statement: str) -> tuple[DDLType | None, str | None]:
    patterns: list[tuple[Pattern[str], DDLType]] = [
        (re.compile(r"\s*create\s+database\s+(?:if\s+not\s+exists\s+)`?([a-zA-Z0-9_]+)`?\s*", re.IGNORECASE), DDLType.CREATEDATABASE),
        (re.compile(r"\s*alter\s+database\s+`?([a-zA-Z0-9_]+)`?\s*", re.IGNORECASE), DDLType.ALTERDATABASE),
        (re.compile(r"\s*drop\s+database\s+(?:if\s+exists\s+)`?([a-zA-Z0-9_]+)`?\s*", re.IGNORECASE), DDLType.DROPDATABASE),
        (re.compile(r"create\s+table\s+(?:`?([a-zA-Z0-9_]+)`?\.)?`?[a-zA-Z0-9_]+`?\s+.*", re.IGNORECASE), DDLType.CREATETABLE),
        (re.compile(r"\s*alter\s+table\s+(?:`?([a-zA-Z0-9_]+)`?\.)?`?[a-zA-Z0-9_]+`?\s+.*", re.IGNORECASE), DDLType.ALTERTABLE),
        (re.compile(r"\s*drop\s+table\s+(?:`?([a-zA-Z0-9_]+)`?\.)?`?[a-zA-Z0-9_]+`?\s*", re.IGNORECASE), DDLType.DROPTABLE),
        (re.compile(r"\s*rename\s+table\s+(?:`?([a-zA-Z0-9_]+)`?\.)?`?[a-zA-Z0-9_]+`?\s+.*", re.IGNORECASE), DDLType.RENAMETABLE),
        (re.compile(r"\s*truncate\s+table\s+(?:`?([a-zA-Z0-9_]+)`?\.)?`?[a-zA-Z0-9_]+`?\s*", re.IGNORECASE), DDLType.TRUNCATETABLE),
        (re.compile(r"\s*create\s+index\s+(?:`?([a-zA-Z0-9_]+)`?\.)?`?[a-zA-Z0-9_]+`?\s+.*", re.IGNORECASE), DDLType.CREATEINDEX),
        (re.compile(r"\s*drop\s+index\s+(?:`?([a-zA-Z0-9_]+)`?\.)?`?[a-zA-Z0-9_]+`?\s*", re.IGNORECASE), DDLType.DROPINDEX),
        (re.compile(r"\s*alter\s+index\s+(?:`?([a-zA-Z0-9_]+)`?\.)?`?[a-zA-Z0-9_]+`?\s+.*", re.IGNORECASE), DDLType.ALTERINDEX),
    ]
    for regex, ddl_type in patterns:
        match = regex.search(statement)
        if match:
            groups = match.groups()
            # print(groups, statement)
            if ddl_type in {DDLType.CREATEDATABASE, DDLType.DROPDATABASE}:
                return ddl_type, groups[0]
            elif len(groups) == 1:
                return ddl_type, groups[0]

    return None, None


class MysqlClient:

    def __init__(self, connection_settings) -> None:
        self.con: MySQLConnection = mysql.connector.connect(**connection_settings)
        self.cur: MySQLCursor = self.con.cursor()
        self.con.autocommit = False
        self.dml_cnt = 0
        self.statement_container = []

    def __c_con(self):
        self.con = mysql.connector.connect(**{"host": "db2", "port": 3306, "user": "root", "passwd": "root_password"})

    def __c_cur(self):
        self.cur: MySQLCursor = self.con.cursor()

    def push_begin(self):
        self.con.start_transaction()
        self.cur: MySQLCursor = self.con.cursor()

    def push_dml(self, sql_text: str, params: tuple) -> None:
        self.statement_container.append((sql_text, params))
        self.dml_cnt += 1
        xxx = self.cur.execute(sql_text, params)

    def push_nondml(self, db: str | None, sql_text: str) -> None:
        # print(f"exec {db} {sql_text}")
        try:
            if db:
                self.con.database = db
            with self.con.cursor() as cur:

                #     # print("ccc########c ", cf)
                cur.execute(sql_text)
            #     print(cur.fetchall())
        except Exception as e:
            print(f"error push_nondml {db} {sql_text} ", e)

    def push_commit(self) -> None:
        # self.dml_container.clear()
        if len(self.statement_container) >= 1:
            self.con.commit()
            self.dml_cnt = 0
            self.statement_container.clear()
            if self.cur:
                self.cur.close()
            self.__c_cur()

    def get_gtid(self, server_uuid: str):
        with self.con.cursor() as cur:
            cur.execute("show master status")


class Checkpoint:
    def push_gtid(gtid):
        pass

    def push_timestamp(ts):
        pass


def set_col(colums: list[Column], col_vals: any):
    for i, col in enumerate(colums):
        if col.type == 16:  # bit
            binary_string = col_vals[i]
            binary_integer = int(binary_string, 2)
            binary_data = binary_integer.to_bytes((binary_integer.bit_length() + 7) // 8, byteorder="big")
            col_vals[i] = binary_data
        elif col.type == 248:  # set
            col_vals[i] = ",".join(col_vals[i])
        else:
            continue


class MysqlReplication:
    def __init__(self, connection_settings: dict, server_id: int, report_slave: str, slave_heartbeat: int, blocking: bool) -> None:
        self.binlogeventstream: Iterator[BinLogEvent] = BinLogStreamReader(
            connection_settings=connection_settings,
            server_id=server_id,
            ignored_events=[RowsQueryLogEvent],
            blocking=blocking,
            report_slave=report_slave,
            slave_heartbeat=slave_heartbeat,
        )

    # def get_binlog_stream() -> Iterator[BinLogEvent]:
    #     mysql_settings = {"host": "127.0.0.1", "port": 3306, "user": "root", "passwd": ""}
    #     stream = BinLogStreamReader(connection_settings=mysql_settings, server_id=3, blocking=True)
    #     return stream

    def __handle_table_map_event(self, event: TableMapEvent):
        pass
        # print("handle_table_map_event", event.schema, event.table)

    def __handle_event_update_rows(self, event: UpdateRowsEvent):
        for row in event.rows:
            keys_list = [key for key in row["after_values"]]
            vals_list = [row["after_values"][key] for key in keys_list]

            keys2_list = [key for key in row["before_values"]]
            vals2_list = [row["before_values"][key] for key in keys2_list]

            set_step = ", ".join([f"{item}=%s" for item in keys_list])

            where_step = f"`{event.primary_key}`=%s" if event.primary_key else set_step

            set_col(event.columns, vals_list)

            set_params = vals_list

            where_parsms = [row["after_values"][event.primary_key]] if event.primary_key else vals2_list

            params = tuple(set_params + where_parsms)
            sql = f"UPDATE `{event.schema}`.`{event.table}` SET {set_step} WHERE {where_step}"
            # print(sql, params)

            # self.mysqlclient.push_dml(sql, params)

            # self.sqlqueue.append(SqlUnit("dml", sql, params))

            # self.sqlunit.kind = "dml"
            # self.sqlunit.sqls.append(tuple(sql, params))
            return OperationDML(sql, params)

    def __handle_event_write_rows(self, event: WriteRowsEvent):
        col: Column = event.columns[1]

        for row in event.rows:
            keys_list = [key for key in row["values"]]
            vals_list = [row["values"][key] for key in keys_list]

            set_col(event.columns, vals_list)

            # for i in range(len(vals_list)):
            #     if type(vals_list[i]) == set:
            #         vals_list[i] = ", ".join(vals_list[i])

            # set type
            # vals_list[-1] = ",".join(map(str, vals_list[-1]))

            # vals_list[9] = int(vals_list[9], 2)

            result = generate_question_marks(len(keys_list))

            sql = f"INSERT INTO `{event.schema}`.`{event.table}`({", ".join(keys_list)}) VALUES({result})"
            # print("sql statement", len(keys_list), len(vals_list), sql, tuple(vals_list))
            # self.sqlqueue.append(SqlUnit("dml", sql, tuple(vals_list)))

            x: tuple[str, tuple] = (sql, tuple(vals_list))
            # print(x)

            # self.mysqlclient.push_dml(sql, tuple(vals_list))

            # self.sqlunit.sqls.append(x)
            return OperationDML(sql, tuple(vals_list))

    def __handle_event_delete_rows(self, event: DeleteRowsEvent):
        for row in event.rows:
            keys_list = [key for key in row["values"]]
            vals_list = [row["values"][key] for key in keys_list]

            set_step = ", ".join([f"{item}=%s" for item in keys_list])

            where_step = f"`{event.primary_key}`=%s" if event.primary_key else set_step

            where_parsms = [row["values"][event.primary_key]] if event.primary_key else vals_list

            params = tuple(where_parsms)
            sql = f"DELETE FROM `{event.schema}`.`{event.table}` WHERE {where_step}"
            # self.sqlqueue.append(SqlUnit("dml", sql, params))

            # self.sqlunit.kind = "dml"
            # self.sqlunit.sqls.append(tuple(sql, params))
            # self.mysqlclient.push_dml(sql, params)
            return OperationDML(sql, params)

    def __handle_event_gtid(self, event: GtidEvent):

        # print(event.last_committed, event.sequence_number, event.gtid)
        # self.operations.append(event.gtid)
        return OperationGtid(event.gtid, event.last_committed)

        # self.sqlunit = SqlUnit("", [event.gtid], [])

        # for sql in self.sqlqueue:
        #     print(self.logfile, event.packet.log_pos, sql)
        # self.sqlqueue.clear()

    def __handle_event_rotate(self, event: RotateEvent):
        # print(event.next_binlog)
        self.logfile = event.next_binlog

    def __handle_event_xid(self, event: XidEvent):
        # for sql, params in self.sqlunit.sqls:
        #     print(self.sqlunit.gtid, self.logfile, event.packet.log_pos, sql, params)
        # # self.sqlqueue.clear()

        return OperationCommit()

    def __handle_event_query(self, event: QueryEvent):
        # print(event.query)
        if event.query == "BEGIN":
            return OperationBegin()
            # print("dml", event.schema)
        else:
            # print(event.schema, event.schema_length, event.query)
            # self.sqlunit.kind = "ddl"
            # self.sqlunit.sqls.append((event.query, ()))
            # print("push non-dml to target mysql", event.schema, event.query)
            ddltype, db = extract_schema(event.query)

            if ddltype and ddltype in (DDLType.CREATEDATABASE, DDLType.DROPDATABASE, DDLType.ALTERDATABASE):

                return OperationDDL(None, event.query)
            else:
                if event.schema_length >= 1:
                    schema: bytes = event.schema
                    # self.mysqlclient.push_nondml(schema.decode("utf-8"), event.query)
                    return OperationDDL(schema.decode("utf-8"), event.query)
                else:
                    if db:
                        return OperationDDL(db, event.query)
                    else:
                        raise Exception("db not know")

            # for sql, params in self.sqlunit.sqls:
            #     print(self.sqlunit.gtid, self.logfile, event.packet.log_pos, sql, params)

    def __handle_event_heartheatlog(self, event: HeartbeatLogEvent):
        return OperationHeartbeat()
        # print(event.__dict__)

    def operation_stream(self):
        handlers = {
            TableMapEvent: self.__handle_table_map_event,
            UpdateRowsEvent: self.__handle_event_update_rows,
            WriteRowsEvent: self.__handle_event_write_rows,
            DeleteRowsEvent: self.__handle_event_delete_rows,
            GtidEvent: self.__handle_event_gtid,
            XidEvent: self.__handle_event_xid,
            QueryEvent: self.__handle_event_query,
            RotateEvent: self.__handle_event_rotate,
            HeartbeatLogEvent: self.__handle_event_heartheatlog,
        }

        for binlogevent in self.binlogeventstream:
            # last_log_file = self.binlogeventstream.log_file
            # last_log_pos = self.binlogeventstream.log_pos
            # print(last_log_file, last_log_pos)
            handler = handlers.get(type(binlogevent))
            if handler:
                operation = handler(binlogevent)
                if operation:
                    yield operation


class MysqlSync:
    # root/root@password@db1:3306?charset=utf8mb4
    # root/root@password@db2:3306?charset=utf8mb4

    def __init__(
        self,
        mysql_source_connection_settings: dict,
        mysql_source_server_id: int,
        mysql_source_report_slave: str,
        mysql_source_slave_heartbeat: int,
        mysql_source_blocking: bool,
        mysql_target_connection_settings: dict,
    ) -> None:
        self.mr = MysqlReplication(
            connection_settings=mysql_source_connection_settings,
            server_id=mysql_source_server_id,
            report_slave=mysql_source_report_slave,
            slave_heartbeat=mysql_source_slave_heartbeat,
            blocking=mysql_source_blocking,
        )
        self.mc = MysqlClient(mysql_target_connection_settings)

    def run(self):
        last_committed = -1
        is_begin = False
        allow_commit = False
        last_operation = None
        last_gtid = {}
        for operation in self.mr.operation_stream():

            if type(operation) == OperationDDL:
                if is_begin and allow_commit:
                    self.mc.push_commit()
                    is_begin = False
                    allow_commit = False
                self.mc.push_nondml(operation.schema, operation.sql_text)
            elif type(operation) == OperationDML:

                self.mc.push_dml(operation.sql_text, operation.params)
            elif type(operation) == OperationBegin:
                if is_begin == False:
                    self.mc.push_begin()
                    is_begin = True
            elif type(operation) == OperationCommit:
                allow_commit = True
            elif type(operation) == OperationGtid:
                if last_committed != operation.last_committed:
                    if is_begin and allow_commit:
                        self.mc.push_commit()
                        is_begin = False
                        allow_commit = False
                    last_committed = operation.last_committed

            elif type(operation) == OperationHeartbeat:
                if type(operation) == type(last_operation):
                    if is_begin and allow_commit:
                        self.mc.push_commit()
                        is_begin = False
                        allow_commit = False
            else:
                pass

            last_operation = type(operation)

        # self.mc.push_commit()


@dataclass
class Config:

    mysql_source_connection_string: str
    mysql_source_server_id: int
    mysql_source_report_slave: str
    mysql_source_slave_heartbeat: int
    mysql_source_blocking: bool
    mysql_target_connection_string: str


parser = argparse.ArgumentParser(prog="ProgramName", description="What the program does", epilog="Text at the bottom of help")
parser.add_argument("--mysql_source_connection_string", type=str, required=True)
parser.add_argument("--mysql_source_server_id", type=int, required=True)
parser.add_argument("--mysql_source_report_slave", type=str, required=True)
parser.add_argument("--mysql_source_slave_heartbeat", type=int, required=True)
parser.add_argument("--mysql_source_blocking", action="store_true")

parser.add_argument("--mysql_target_connection_string", type=str, required=True)
args = parser.parse_args()
config = Config(**vars(args))
print(config)

MysqlSync(
    parse_connection_string(config.mysql_source_connection_string),
    config.mysql_source_server_id,
    config.mysql_source_report_slave,
    config.mysql_source_slave_heartbeat,
    config.mysql_source_blocking,
    parse_connection_string(config.mysql_target_connection_string),
).run()
