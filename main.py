import argparse
import enum
import importlib.util
import json
import logging
import os
import re
import sys
import time
from collections.abc import Iterator
from dataclasses import dataclass
from typing import Pattern

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

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)
# logger = logging.getLogger("mysqlsync")
# logger.setLevel(logging.DEBUG)
# handler = logging.StreamHandler()
# handler.setLevel(logging.DEBUG)

# formatter = logging.Formatter('%(asctime)s - %(name)s - %(levelname)s - %(message)s')
# handler.setFormatter(formatter)

# # 添加处理器到 logger
# logger.addHandler(handler)
# logging = logger


# signal.signal(signal.SIGINT, handle_exit_signal)
# signal.signal(signal.SIGTERM, handle_exit_signal)


# def generate_question_marks(n):
#     return ",".join(f'{"%s"}' for _ in range(n))


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
    # DROPINDEX = enum.auto()


class OperationType(enum.Enum):
    DDL = enum.auto()
    BEGIN = enum.auto()
    COMMIT = enum.auto()
    DML = enum.auto()
    GTID = enum.auto()


@dataclass
class OperationNonDML:
    oper_type: DDLType
    schema: str | None
    table: str | None
    query: str


@dataclass
class OperationDML:
    schema: str
    table: str


@dataclass
class OperationDMLInsert(OperationDML):
    values: dict
    primary_key: str | tuple | None


@dataclass
class OperationDMLDelete(OperationDML):
    values: dict
    primary_key: str | tuple | None


@dataclass
class OperationDMLUpdate(OperationDML):
    after_values: dict
    before_values: dict
    primary_key: str | tuple | None


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
    server_uuid: str
    xid: int
    last_committed: int


def parse_connection_string(conn_str: str) -> dict:
    result = {}

    user_pass_part, host_part = conn_str.split("@")

    user, passwd = user_pass_part.split("/")
    result["user"] = user
    result["password"] = passwd

    if "?" in host_part:
        host_port, params = host_part.split("?")

        params_dict = {}
        for param in params.split("&"):
            key, value = param.split("=")
            if key == "compress":
                if value.lower() == "true":
                    params_dict[key] = True
                elif value.lower() == "false":
                    params_dict[key] = False
                else:
                    raise Exception(
                        f"connection arguments key 'compress', value '{value}' not support."
                    )
            elif key == "charset":
                params_dict[key] = value
            else:
                raise Exception(f"connection arguments key '{key}' not support.")

        result.update(params_dict)
    else:
        host_port = host_part

    host, port = host_port.split(":")
    result["host"] = host
    result["port"] = int(port)

    result.setdefault("charset", "utf8mb4")

    return result


def dmlOperation2Sql(operation: OperationDML) -> tuple[str, tuple]:
    if isinstance(operation, OperationDMLInsert):
        values = (
            operation.values
            if isinstance(operation, OperationDMLInsert)
            else operation.after_values
        )
        keys = ", ".join(values.keys())
        values_placeholder = ",".join(f'{"%s"}' for _ in range(len(values)))
        sql = f"REPLACE INTO `{operation.schema}`.`{operation.table}`({keys}) VALUES({values_placeholder})"
        params = tuple(values.values())
    elif isinstance(operation, OperationDMLDelete):
        if operation.primary_key:
            if isinstance(operation.primary_key, tuple):
                condition = " AND ".join([f"`{k}` = %s" for k in operation.primary_key])
            else:
                condition = f"`{operation.primary_key}` = %s"
            primary_values = (
                (operation.values[k] for k in operation.primary_key)
                if isinstance(operation.primary_key, tuple)
                else (operation.values[operation.primary_key],)
            )
        else:
            condition = " AND ".join([f"`{k}` = %s" for k in operation.values.keys()])
            primary_values = tuple(operation.values.values())
        sql = f"DELETE FROM `{operation.schema}`.`{operation.table}` WHERE {condition}"
        params = tuple(primary_values)
    elif isinstance(operation, OperationDMLUpdate):
        set_clause = ", ".join([f"{k} = %s" for k in operation.after_values.keys()])
        if operation.primary_key:
            if isinstance(operation.primary_key, tuple):
                condition = " AND ".join([f"`{k}` = %s" for k in operation.primary_key])
            else:
                condition = f"`{operation.primary_key}` = %s"
            primary_values = (
                (operation.before_values[k] for k in operation.primary_key)
                if isinstance(operation.primary_key, tuple)
                else (operation.before_values[operation.primary_key],)
            )
        else:
            condition = " AND ".join(
                [f"`{k}` = %s" for k in operation.before_values.keys()]
            )
            primary_values = tuple(operation.before_values.values())
        sql = f"UPDATE `{operation.schema}`.`{operation.table}` SET {set_clause} WHERE {condition}"
        params = tuple(operation.after_values.values()) + tuple(primary_values)
    return sql, params


nondml_patterns: list[tuple[Pattern[str], DDLType]] = [
    (
        re.compile(
            r"\s*create\s+database\s+(?:if\s+not\s+exists\s+)?`?(\w+)`?\s*",
            re.IGNORECASE,
        ),
        DDLType.CREATEDATABASE,
    ),
    (
        re.compile(r"\s*alter\s+database\s+`?(\w+)`?\s*", re.IGNORECASE),
        DDLType.ALTERDATABASE,
    ),
    (
        re.compile(
            r"\s*drop\s+database\s+(?:if\s+exists\s+)?`?(\w+)`?\s*", re.IGNORECASE
        ),
        DDLType.DROPDATABASE,
    ),
    (
        re.compile(
            r"\s*create\s+table\s+(?:`?(\w+)`?\.)?`?(\w+)`?\s?.*", re.IGNORECASE
        ),
        DDLType.CREATETABLE,
    ),
    (
        re.compile(r"\s*alter\s+table\s+(?:`?(\w+)`?\.)?`?(\w+)`?\s+.*", re.IGNORECASE),
        DDLType.ALTERTABLE,
    ),
    (
        re.compile(r"\s*drop\s+table\s+(?:`?(\w+)`?\.)?`?(\w+)`?\s*", re.IGNORECASE),
        DDLType.DROPTABLE,
    ),
    (
        re.compile(
            r"\s*rename\s+table\s+(?:`?(\w+)`?\.)?`?(\w+)`?\s+.*", re.IGNORECASE
        ),
        DDLType.RENAMETABLE,
    ),
    (
        re.compile(
            r"\s*truncate\s+table\s+(?:`?(\w+)`?\.)?`?(\w+)`?\s*", re.IGNORECASE
        ),
        DDLType.TRUNCATETABLE,
    ),
    (
        re.compile(
            r"\s*create\s+index\s+\w+\s+on\s+(?:`?(\w+)`?\.)?`?(\w+)`?\s?.*",
            re.IGNORECASE,
        ),
        DDLType.CREATEINDEX,
    ),
]


def extract_schema(statement: str) -> tuple[DDLType | None, str | None, str | None]:
    for regex, ddl_type in nondml_patterns:
        match = regex.search(statement)
        if match:
            groups = match.groups()
            if ddl_type in {
                DDLType.CREATEDATABASE,
                DDLType.ALTERDATABASE,
                DDLType.DROPDATABASE,
            }:
                # return ddl_type, groups[0], None
                return ddl_type, None, None
            elif ddl_type in {
                DDLType.CREATETABLE,
                DDLType.ALTERTABLE,
                DDLType.DROPTABLE,
                DDLType.TRUNCATETABLE,
                DDLType.RENAMETABLE,
                DDLType.CREATEINDEX,
            }:
                return ddl_type, groups[0], groups[1]

    return None, None, None


class MysqlClient:
    def __init__(self, settings) -> None:
        self.con: MySQLConnection = mysql.connector.connect(
            **settings["connection_settings"]
        )
        self.cur: MySQLCursor = self.con.cursor()
        self.con.autocommit = False
        self.dml_cnt = 0
        self.active_trx = False

    def __c_con(self):
        pass
        # self.con = mysql.connector.connect(**{"host": "db2", "port": 3306, "user": "root", "passwd": "root_password"})

    def __c_cur(self):
        self.cur: MySQLCursor = self.con.cursor()

    # def get_gtidset(self):
    #     with self.con.cursor() as cur:
    #         cur.execute("show master status")
    #         _, _, _, _, gtidset = cur.fetchone()
    #         return gtidset

    def push_begin(self):
        if self.active_trx:
            return
        self.con.start_transaction()
        self.active_trx = True
        # self.cur: MySQLCursor = self.con.cursor()

    def push_dml(self, sql_text: str, params: tuple) -> None:
        self.cur.execute(sql_text, params)
        self.dml_cnt += 1

    def push_nondml(self, db: str | None, sql_text: str) -> None:
        if db:
            self.con.database = db
        with self.con.cursor() as cur:
            cur.execute(sql_text)

        self.active_trx = False

    def push_commit(self) -> None:
        if self.dml_cnt >= 1 and self.active_trx:
            self.con.commit()
        else:
            self.con.rollback()
        self.dml_cnt = 0
        self.active_trx = False
        # if self.cur:
        #     self.cur.close()
        # self.__c_cur()

    def push_rollback(self) -> None:
        self.con.rollback()

    def get_gtid(self, server_uuid: str):
        with self.con.cursor() as cur:
            cur.execute("show master status")


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
        binary_data = binary_integer.to_bytes(
            (binary_integer.bit_length() + 7) // 8, byteorder="big"
        )
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
    def __init__(self, source_settings: dict) -> None:
        self.binlogeventstream: Iterator[BinLogEvent] = BinLogStreamReader(
            **source_settings
        )

    def __handle_table_map_event(self, event: TableMapEvent):
        pass

    def __handle_event_update_rows(self, event: UpdateRowsEvent) -> list[OperationDML]:
        ls = []
        for row in event.rows:
            new_after_values = reset_values(row["after_values"], event.columns)
            new_before_values = reset_values(row["before_values"], event.columns)
            ls.append(
                OperationDMLUpdate(
                    event.schema,
                    event.table,
                    new_after_values,
                    new_before_values,
                    event.primary_key,
                )
            )
        return ls

    def __handle_event_write_rows(
        self, event: WriteRowsEvent
    ) -> list[list[OperationDML]]:
        ls = []
        for row in event.rows:
            new_values = reset_values(row["values"], event.columns)
            ls.append(
                OperationDMLInsert(
                    event.schema, event.table, new_values, event.primary_key
                )
            )
        return ls

    def __handle_event_delete_rows(
        self, event: DeleteRowsEvent
    ) -> list[list[OperationDML]]:
        ls = []
        for row in event.rows:
            new_values = reset_values(row["values"], event.columns)
            ls.append(
                OperationDMLDelete(
                    event.schema, event.table, new_values, event.primary_key
                )
            )
        return ls

    def __handle_event_gtid(self, event: GtidEvent):
        server_uuid, xid = event.gtid.split(":")
        return OperationGtid(server_uuid, xid, event.last_committed)

    def __handle_event_rotate(self, event: RotateEvent):
        self.logfile = event.next_binlog

    def __handle_event_xid(self, event: XidEvent):
        return OperationCommit()

    def __handle_event_query(self, event: QueryEvent):
        if event.query == "BEGIN":
            return OperationBegin()
        elif event.query == "COMMIT":
            logging.warning('empty trx, query:"COMMIT"')
            return None
        else:
            ddltype, db, tab = extract_schema(event.query)

            if ddltype is None:
                raise Exception("not know queryevent query", event.query)

            if ddltype in (
                DDLType.CREATEDATABASE,
                DDLType.DROPDATABASE,
                DDLType.ALTERDATABASE,
            ):
                return OperationNonDML(ddltype, None, None, event.query)

            else:
                if ddltype in (
                    DDLType.CREATETABLE,
                    DDLType.ALTERTABLE,
                    DDLType.TRUNCATETABLE,
                    DDLType.CREATEINDEX,
                    DDLType.RENAMETABLE,
                    DDLType.DROPTABLE,
                ):
                    schema = (
                        event.schema.decode("utf-8") if event.schema_length >= 1 else db
                    )

                    if schema:
                        return OperationNonDML(ddltype, schema, tab, event.query)
                    else:
                        raise Exception("db not know", event.__dict__)

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

        log_pos = 4
        end_log_pos = 0
        for binlogevent in self.binlogeventstream:
            if type(binlogevent) != HeartbeatLogEvent:
                log_file: str = self.binlogeventstream.log_file
                end_log_pos: int = self.binlogeventstream.log_pos
            handler = handlers.get(type(binlogevent))
            if handler:
                operation = handler(binlogevent)
                if operation:
                    if isinstance(operation, list):
                        for o in operation:
                            yield log_file, log_pos, binlogevent.timestamp, o
                    else:
                        yield log_file, log_pos, binlogevent.timestamp, operation
            log_pos = end_log_pos


def parse_gtidset(gtidset_str: str) -> dict[str, int]:
    gtid_re = re.compile(
        r"^([a-zA-Z0-9]{8}-[a-zA-Z0-9]{4}-[a-zA-Z0-_]{4}-[a-zA-Z0-9]{4}-[a-zA-Z0-9]{12}):[0-9]+-([0-9]+)$"
    )

    _gtidset = {}
    for gtid in gtidset_str.split(","):
        match = gtid_re.search(gtid)
        if match:
            server_uuid, xid = match.groups()
            _gtidset[server_uuid] = int(xid)
        else:
            raise Exception(f"gtid set format error {gtid}")
    return _gtidset


def format_filter_tables(tables: list[str]) -> dict:
    filter_dict: dict[str, list[str]] = {}
    for dbtab in tables:
        db, tab = dbtab.split(".")
        if db in filter_dict:
            filter_dict[db].append(tab)
        else:
            filter_dict[db] = [tab]
    return filter_dict


class MysqlRowCompare:
    def __init__(
        self,
        mysql_source_connection_settings: dict,
        mysql_target_connection_settings: dict,
    ):
        pass

    def compare_key(self, dbtab: str, primarykey: dict[str, any]):
        pass

    def compare_nonkey(self, dbtab: str, values: list[any]):
        pass


class Checkpoint:
    def __init__(self) -> None:
        self.checkpoint_gtidset: str = self.__get_checkpoint_gtidset()

    def persist_checkpoint_gitdset(self, gtidset: dict[str, int]):
        print("checkpoint", gtidset)
        with open("ckpt.json", "w", encoding="utf8") as f:
            return json.dump(gtidset, f)

    def __get_checkpoint_gtidset(self) -> str:
        start_gtidset = {}
        if os.path.exists("ckpt.json"):
            with open("ckpt.json", "r", encoding="utf8") as f:
                _gtidset: dict[str, int] = json.load(f)
                for _server_uuid, _xid in _gtidset.items():
                    start_gtidset[_server_uuid] = f"1-{_xid}"
        return start_gtidset


class MysqlSync:
    def __init__(
        self,
        mysql_source_settings: dict,
        mysql_target_settings: dict,
        mysql_sync_settings: dict,
    ) -> None:
        self.args_mysql_sync_merge_trx = mysql_sync_settings["merge_trx"]

        self.ckpt = Checkpoint()

        print(f"start {self.ckpt.checkpoint_gtidset}")
        if self.ckpt:
            mysql_source_settings["auto_position"] = self.ckpt.checkpoint_gtidset

        self.mr = MysqlReplication(mysql_source_settings)
        self.mc = MysqlClient(mysql_target_settings)

        self.statistics = {
            "dml": 0,
            "nomdml": 0,
            "trx-merge": 0,
            "trx": 0,
            "dml-delete": 0,
            "dml-insert": 0,
            "dml-update": 0,
        }

        self.last_committed = -1
        self.allow_commit = False
        self.last_operation = None

        self.allow_ckeckpoint = False
        self.checkpoint_gtidset = self.ckpt.checkpoint_gtidset
        self.checkpoint_position = ()

        self.latest_gtidset = {}
        self.latest_position = ()

        self.exclude_tables: dict[str, list[str]] = {}

    def __merge_commit(self):
        if self.allow_commit:
            self.mc.push_commit()
            self.allow_commit = False
            self.statistics["trx-merge"] += 1
            self.checkpoint_gtidset = self.latest_gtidset
            self.checkpoint_position = self.latest_position
            if self.allow_ckeckpoint:
                self.ckpt.persist_checkpoint_gitdset(self.checkpoint_gtidset)
                self.allow_ckeckpoint = False

    def run(self):
        _ts_0 = time.time()
        _timestamp = 0
        _error = False

        for log_file, log_pos, timestamp, operation in self.mr.operation_stream():
            if type(operation) == OperationNonDML:
                self.__merge_commit()

                try:
                    self.mc.push_nondml(operation.schema, operation.query)

                    self.statistics["nomdml"] += 1
                    nondml_name = f"nondml-{operation.oper_type.name.lower()}"
                    self.statistics[nondml_name] = (
                        self.statistics.get(nondml_name, 0) + 1
                    )
                except Exception as e:
                    logging.error(
                        f"error push_nondml {operation.schema} {operation.query} {e}"
                    )
                    sys.exit(1)
                logging.info(
                    f"nondml '{nondml_name}' schema: '{operation.schema}' query: '{operation.query}'"
                )
            elif isinstance(operation, OperationDML):
                if (
                    operation.schema in self.exclude_tables
                    and operation.table in self.exclude_tables[operation.schema]
                ):
                    logging.info(
                        f"exclude tables `{operation.schema}`.`{operation.table}`"
                    )
                    continue

                self.statistics["dml"] += 1
                if type(operation) == OperationDMLDelete:
                    self.statistics["dml-delete"] += 1
                elif type(operation) == OperationDMLInsert:
                    self.statistics["dml-insert"] += 1
                elif type(operation) == OperationDMLUpdate:
                    self.statistics["dml-update"] += 1

                try:
                    sql_text, params = dmlOperation2Sql(operation)
                    self.mc.push_dml(sql_text, params)
                except Exception as e:
                    print("dml error", operation, e)
                    raise e
            elif type(operation) == OperationBegin:
                self.mc.push_begin()
            elif type(operation) == OperationCommit:
                self.statistics["trx"] += 1
                if self.args_mysql_sync_merge_trx:
                    self.allow_commit = True
                else:
                    self.mc.push_commit()
            elif type(operation) == OperationGtid:
                if self.args_mysql_sync_merge_trx:
                    if self.last_committed != operation.last_committed:
                        self.__merge_commit()
                        self.last_committed = operation.last_committed

                self.latest_gtidset[operation.server_uuid] = int(operation.xid)
                self.latest_position = (log_file, log_pos)
            elif (
                type(operation) == OperationHeartbeat
                and type(self.last_operation) == OperationHeartbeat
            ):
                _timestamp = time.time()
                self.__merge_commit()
            else:
                pass

            if type(operation) != OperationHeartbeat:
                _timestamp = timestamp

            _ts_1 = time.time()
            if _ts_1 >= _ts_0 + 10:
                print(self.statistics)
                x = ",".join(
                    [f"{key}:{value}" for key, value in self.checkpoint_gtidset.items()]
                )
                fx = f"syncinfo dml:{self.statistics["dml"]} nondml:{self.statistics["nomdml"]} trx-merge:{self.statistics["trx-merge"]} trx:{self.statistics["trx"]} gtidset:"
                # fx = f"syncinfo dml:{self.statistics['dml']} nondml:{self.statistics['nomdml']}"
                print()
                print(f"delay {int(time.time() - _timestamp)}")

                _ts_0 = _ts_1

                if self.allow_commit and not self.allow_ckeckpoint:
                    self.allow_ckeckpoint = True

            self.last_operation = operation

        self.__merge_commit()

        print(self.statistics)
        print(
            f"syncinfo dml:{self.statistics["dml"]} nondml:{self.statistics["nomdml"]} trx-merge:{self.statistics["trx-merge"]} trx:{self.statistics["trx"]} gtidset:{','.join([f"{key}:{value}" for key, value in self.checkpoint_gtidset.items()])}"
        )


# @dataclass
# class Config:
#     mysql_source_connection_string: str
#     mysql_source_server_id: int
#     mysql_source_report_slave: str
#     mysql_source_slave_heartbeat: int
#     mysql_source_blocking: bool
#     mysql_source_exclude_gtids: str | None
#     # mysql_source_logfile_position: str | None
#     mysql_source_filter_tables: list[str] | None
#     mysql_target_connection_string: str | None
#     mysql_sync_force_idempotent: bool
#     mysql_sync_merge_trx: bool


# parser = argparse.ArgumentParser(
#     prog="MysqlSync",
#     description="What the program does",
#     epilog="Text at the bottom of help",
# )
# parser.add_argument("--mysql_source_connection_string", type=str, required=True)
# parser.add_argument("--mysql_source_server_id", type=int, required=True)
# parser.add_argument("--mysql_source_report_slave", type=str, required=True)
# parser.add_argument("--mysql_source_slave_heartbeat", type=int, required=True)
# parser.add_argument("--mysql_source_blocking", action="store_true")
# parser.add_argument(
#     "--mysql_source_exclude_gtids",
#     type=str,
#     required=True,
#     help="Specify GTIDs to exclude.",
# )
# parser.add_argument("--mysql_source_filter_tables", type=str, nargs="+", default=[])

# parser.add_argument("--mysql_target_connection_string", type=str, required=True)

# parser.add_argument("--mysql_sync_force_idempotent", action="store_true")
# parser.add_argument("--mysql_sync_merge_trx", action="store_true")

# args = parser.parse_args()
# config = Config(**vars(args))
# print(config)


# MysqlSync(
#     parse_connection_string(config.mysql_source_connection_string),
#     config.mysql_source_server_id,
#     config.mysql_source_report_slave,
#     config.mysql_source_slave_heartbeat,
#     config.mysql_source_blocking,
#     config.mysql_source_exclude_gtids,
#     config.mysql_source_filter_tables,
#     parse_connection_string(config.mysql_target_connection_string),
#     config.mysql_sync_force_idempotent,
#     config.mysql_sync_merge_trx,
# ).run()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--config", required=True, help="Path to the configuration file"
    )
    args = parser.parse_args()
    if args.config:
        spec = importlib.util.spec_from_file_location("settings", args.config)
        settings = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(settings)

        print(f"source settings: {settings.source_settings}")
        print(f"target settings: {settings.target_settings}")
        print(f"sync settings: {settings.sync_settings}")

        MysqlSync(
            settings.source_settings, settings.target_settings, settings.sync_settings
        ).run()


if __name__ == "__main__":
    main()
