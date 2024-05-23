import argparse
import enum
import re
import time
from collections import deque
from collections.abc import Iterator
from dataclasses import dataclass
from typing import Optional, Pattern, cast

import mysql.connector
from mysql.connector import MySQLConnection
from mysql.connector.cursor import MySQLCursor
from pymysqlreplication import BinLogStreamReader
from pymysqlreplication.column import Column
from pymysqlreplication.constants import FIELD_TYPE
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
    pass


@dataclass
class OperationDMLInsert(OperationDML):
    schema: str
    table: str
    values: dict
    primary_key: str | None


@dataclass
class OperationDMLDelete(OperationDML):
    schema: str
    table: str
    values: dict
    primary_key: str | None


@dataclass
class OperationDMLUpdate(OperationDML):
    schema: str
    table: str
    after_values: dict
    before_values: dict
    primary_key: str | None


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


def deleteOperation2SqlOperation(o: OperationDMLDelete):
    keys_list = [key for key in o.values]
    vals_list = [o.values[key] for key in keys_list]

    set_step = ", ".join([f"{item}=%s" for item in keys_list])

    where_step = f"`{o.primary_key}`=%s" if o.primary_key else set_step

    where_parsms = [o.values[o.primary_key]] if o.primary_key else vals_list

    sql = f"DELETE FROM `{o.schema}`.`{o.table}` WHERE {where_step}"
    return sql, tuple(where_parsms)


def updateOperation2SqlOperation(o: OperationDMLUpdate, replace: bool) -> list[OperationDML]:
    if replace:
        new_keys_list = [key for key in o.after_values]
        new_vals_list = [o.after_values[key] for key in new_keys_list]

        result = generate_question_marks(len(new_keys_list))
        sql = f"REPLACE INTO `{o.schema}`.`{o.table}`({", ".join(new_keys_list)}) VALUES({result})"
        return sql, tuple(new_vals_list)
    else:
        keys_list_after = [key for key in o.after_values]
        vals_list_after = [o.after_values[key] for key in keys_list_after]

        key_list_before = [key for key in o.before_values]
        vals_list_before = [o.before_values[key] for key in key_list_before]

        set_step = ", ".join([f"{item}=%s" for item in keys_list_after])

        where_step = f"`{o.primary_key}`=%s" if o.primary_key else set_step

        set_params = vals_list_after

        where_parsms = [o.after_values[o.primary_key]] if o.primary_key else vals_list_before

        sql = f"UPDATE `{o.schema}`.`{o.table}` SET {set_step} WHERE {where_step}"
        return sql, tuple(set_params + where_parsms)


def writeOperation2SqlOperation(o: OperationDMLInsert, replace: bool) -> list[OperationDML]:
    new_keys_list = [key for key in o.values]
    new_vals_list = [o.values[key] for key in new_keys_list]
    result = generate_question_marks(len(new_keys_list))

    sql = (
        f"REPLACE INTO `{o.schema}`.`{o.table}`({", ".join(new_keys_list)}) VALUES({result})"
        if replace
        else f"INSERT INTO `{o.schema}`.`{o.table}`({", ".join(new_keys_list)}) VALUES({result})"
    )
    return sql, tuple(new_vals_list)


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
        pass
        # self.con = mysql.connector.connect(**{"host": "db2", "port": 3306, "user": "root", "passwd": "root_password"})

    def __c_cur(self):
        self.cur: MySQLCursor = self.con.cursor()

    def get_gtidset(self):
        with self.con.cursor() as cur:
            cur.execute("show master status")
            _, _, _, _, gtidset = cur.fetchone()
            return gtidset

    def push_begin(self):
        self.con.start_transaction()
        self.cur: MySQLCursor = self.con.cursor()

    def push_dml(self, sql_text: str, params: tuple) -> None:
        self.statement_container.append((sql_text, params))
        self.dml_cnt += 1
        self.cur.execute(sql_text, params)

    def push_nondml(self, db: str | None, sql_text: str) -> None:
        try:
            if db:
                self.con.database = db
            with self.con.cursor() as cur:
                cur.execute(sql_text)
        except Exception as e:
            print(f"error push_nondml {db} {sql_text} ", e)

    def push_commit(self) -> None:
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


def reset_col_val(colum_type: int, col_val: any):
    if colum_type in [
        FIELD_TYPE.VAR_STRING,
        FIELD_TYPE.VARCHAR,
        FIELD_TYPE.DECIMAL,
        FIELD_TYPE.CHAR,
        FIELD_TYPE.STRING,
        FIELD_TYPE.SHORT,
        FIELD_TYPE.LONG,
        FIELD_TYPE.INT24,
        FIELD_TYPE.LONGLONG,
        FIELD_TYPE.FLOAT,
        FIELD_TYPE.DOUBLE,
        FIELD_TYPE.NEWDECIMAL,
        FIELD_TYPE.DATE,
        FIELD_TYPE.DATETIME2,
        FIELD_TYPE.TIMESTAMP2,
        FIELD_TYPE.TIME2,
        FIELD_TYPE.YEAR,
        FIELD_TYPE.BLOB,
        FIELD_TYPE.ENUM,
    ]:
        return col_val
    elif colum_type == FIELD_TYPE.BIT:
        binary_string = col_val
        binary_integer = int(binary_string, 2)
        binary_data = binary_integer.to_bytes((binary_integer.bit_length() + 7) // 8, byteorder="big")
        return binary_data
    elif colum_type == FIELD_TYPE.SET:
        return ",".join(col_val)
    else:
        raise Exception(f"unreset col val type {colum_type} {type(col_val)} {col_val}")


def reset_values(values: dict, columns: list[Column]) -> dict[str, any]:
    new_values: dict[str, any] = {}
    for column in columns:
        col_name = column.name
        if col_name in values:
            new_values[col_name] = reset_col_val(column.type, values[col_name])
    return new_values


class MysqlReplication:
    def __init__(
        self, connection_settings: dict, server_id: int, report_slave: str, slave_heartbeat: int, blocking: bool, gtidset: str | None
    ) -> None:
        self.binlogeventstream: Iterator[BinLogEvent] = BinLogStreamReader(
            connection_settings=connection_settings,
            server_id=server_id,
            ignored_events=[RowsQueryLogEvent],
            blocking=blocking,
            report_slave=report_slave,
            slave_heartbeat=slave_heartbeat,
            auto_position=gtidset,
        )

    def __handle_table_map_event(self, event: TableMapEvent):
        pass
        # print("handle_table_map_event", event.schema, event.table)

    def __handle_event_update_rows(self, event: UpdateRowsEvent) -> list[OperationDML]:
        ls = []
        for row in event.rows:
            new_after_values = reset_values(row["after_values"], event.columns)
            new_before_values = reset_values(row["before_values"], event.columns)
            ls.append(OperationDMLUpdate(event.schema, event.table, new_after_values, new_before_values, event.primary_key))
        return ls
        # return [updateRowsEvent2Update(event), updateRowsEvent2Replace(event)]

    def __handle_event_write_rows(self, event: WriteRowsEvent) -> list[list[OperationDML]]:
        ls = []
        for row in event.rows:
            new_values = reset_values(row["values"], event.columns)
            ls.append(OperationDMLInsert(event.schema, event.table, new_values, event.primary_key))
        return ls

    def __handle_event_delete_rows(self, event: DeleteRowsEvent) -> list[list[OperationDML]]:
        ls = []
        for row in event.rows:
            new_values = reset_values(row["values"], event.columns)
            ls.append(OperationDMLDelete(event.schema, event.table, new_values, event.primary_key))
        return ls

    def __handle_event_gtid(self, event: GtidEvent):
        return OperationGtid(event.gtid, event.last_committed)

    def __handle_event_rotate(self, event: RotateEvent):
        self.logfile = event.next_binlog

    def __handle_event_xid(self, event: XidEvent):
        return OperationCommit()

    def __handle_event_query(self, event: QueryEvent):
        if event.query == "BEGIN":
            return OperationBegin()
        else:
            ddltype, db = extract_schema(event.query)

            if ddltype and ddltype in (DDLType.CREATEDATABASE, DDLType.DROPDATABASE, DDLType.ALTERDATABASE):
                return OperationDDL(None, event.query)
            else:
                if event.schema_length >= 1:
                    schema: bytes = event.schema
                    return OperationDDL(schema.decode("utf-8"), event.query)
                else:
                    if db:
                        return OperationDDL(db, event.query)
                    else:
                        raise Exception("db not know")

    def __handle_event_heartheatlog(self, event: HeartbeatLogEvent):
        return OperationHeartbeat()

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
                    if type(operation) == list:
                        for o in operation:
                            yield o
                    else:
                        yield operation


gtid_re = re.compile(r"^([a-zA-Z0-9_]{8}-[a-zA-Z0-9_]{4}-[a-zA-Z0-9_]{4}-[a-zA-Z0-9_]{4}-[a-zA-Z0-9_]{12}):[0-9]+-([0-9]+)$")


def parse_gtidset(gtidset_str: str) -> dict[str, int]:
    _gtidset = {}
    for gtid in gtidset_str.split(","):
        match = gtid_re.search(gtid)
        if match:
            server_uuid, xid = match.groups()
            _gtidset[server_uuid] = int(xid)
        else:
            raise Exception(f"gtid set format error {gtid}")
    return _gtidset


class MysqlRowCompare:
    def __init__(self, mysql_source_connection_settings: dict, mysql_target_connection_settings: dict):
        pass

    def compare_key(self, dbtab: str, primarykey: dict[str, any]):
        pass

    def compare_nonkey(self, dbtab: str, values: list[any]):
        pass


class MysqlSync:
    def __init__(
        self,
        mysql_source_connection_settings: dict,
        mysql_source_server_id: int,
        mysql_source_report_slave: str,
        mysql_source_slave_heartbeat: int,
        mysql_source_blocking: bool,
        mysql_source_gtidset: str | None,
        mysql_target_connection_settings: dict,
        mysql_sync_force_idempotent: bool,
    ) -> None:
        self.mr = MysqlReplication(
            connection_settings=mysql_source_connection_settings,
            server_id=mysql_source_server_id,
            report_slave=mysql_source_report_slave,
            slave_heartbeat=mysql_source_slave_heartbeat,
            blocking=mysql_source_blocking,
            gtidset=mysql_source_gtidset,
        )
        self.mc = MysqlClient(mysql_target_connection_settings)
        self.gtid_sets: dict[str, int] = {}

        self.abc = {"dml": 0, "cmt": 0}

        self.last_committed = -1
        self.is_begin = False
        self.allow_commit = False
        self.last_operation = None

        # self.target_gtidset = parse_gtidset(self.mc.get_gtidset())
        # self.target_idempotent = {server_uuid: True for server_uuid in self.target_gtidset.keys()}
        # self.last_gtidset = {}

    def __update_gtid(self, gtid: str):
        # 73b24aef-0b4d-11ef-9a54-1418774ca835:1-13313466:13313468-15821064:15821066-18093826:18093828-21851433:21851435-24792086:24792088-27279728:27279730-29405984,
        # eb559d55-f6e4-11ee-94e4-c81f66d988c2:21564725-52461618,
        # f62d4ccd-b0f7-11ee-bad1-005056b3c0f8:1-2
        server_uuid, xid = gtid.split(":")
        self.gtid_sets[server_uuid] = int(xid)

    def __commit(self):
        if self.is_begin and self.allow_commit:
            self.mc.push_commit()
            self.abc["cmt"] += 1
            self.is_begin = False
            self.allow_commit = False

    def run(self):
        _ts = time.time()
        idempotent = True

        for operation in self.mr.operation_stream():
            if type(operation) == OperationDDL:
                self.__commit()
                self.mc.push_nondml(operation.schema, operation.sql_text)
            elif isinstance(operation, OperationDML):
                self.abc["dml"] += 1
                if type(operation) == OperationDMLUpdate:
                    self.mc.push_dml(*updateOperation2SqlOperation(operation, True))
                if type(operation) == OperationDMLDelete:
                    self.mc.push_dml(*deleteOperation2SqlOperation(operation))
                if type(operation) == OperationDMLInsert:
                    self.mc.push_dml(*writeOperation2SqlOperation(operation, True))
            elif type(operation) == OperationBegin:
                if self.is_begin == False:
                    self.mc.push_begin()
                    self.is_begin = True
            elif type(operation) == OperationCommit:
                self.allow_commit = True
            elif type(operation) == OperationGtid:
                if self.last_committed != operation.last_committed:
                    self.__commit()
                    self.last_committed = operation.last_committed

                server_uuid, xid = operation.gtid.split(":")
                self.gtid_sets[server_uuid] = int(xid)
            elif type(operation) == OperationHeartbeat:
                if type(self.last_operation) == OperationHeartbeat:
                    self.__commit()
            else:
                pass

            self.last_operation = type(operation)

            _ts_2 = time.time()
            if _ts_2 >= _ts + 10:
                # self.__compare_gtidxid(g)
                # self.__checkpoint()
                print(_ts_2, self.abc)
                print(self.gtid_sets)

                _ts = _ts_2

        if self.is_begin and self.allow_commit:
            self.__commit()

        print(_ts_2, self.abc)
        print(self.gtid_sets)
        # self.mc.push_commit()


@dataclass
class Config:
    mysql_source_connection_string: str
    mysql_source_server_id: int
    mysql_source_report_slave: str
    mysql_source_slave_heartbeat: int
    mysql_source_blocking: bool
    mysql_target_connection_string: str
    mysql_source_gtidset: str | None
    mysql_sync_force_idempotent: bool


parser = argparse.ArgumentParser(prog="ProgramName", description="What the program does", epilog="Text at the bottom of help")
parser.add_argument("--mysql_source_connection_string", type=str, required=True)
parser.add_argument("--mysql_source_server_id", type=int, required=True)
parser.add_argument("--mysql_source_report_slave", type=str, required=True)
parser.add_argument("--mysql_source_slave_heartbeat", type=int, required=True)
parser.add_argument("--mysql_source_blocking", action="store_true")
parser.add_argument("--mysql_source_gtidset", type=str)

parser.add_argument("--mysql_target_connection_string", type=str, required=True)

parser.add_argument("--mysql_sync_force_idempotent", action="store_true")


args = parser.parse_args()

config = Config(**vars(args))

MysqlSync(
    parse_connection_string(config.mysql_source_connection_string),
    config.mysql_source_server_id,
    config.mysql_source_report_slave,
    config.mysql_source_slave_heartbeat,
    config.mysql_source_blocking,
    config.mysql_source_gtidset,
    parse_connection_string(config.mysql_target_connection_string),
    config.mysql_sync_force_idempotent,
).run()
