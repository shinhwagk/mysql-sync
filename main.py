import argparse
import importlib.util
import json
import os
import re
import sys
import time
from collections.abc import Iterator
from dataclasses import asdict, dataclass
from datetime import datetime
from enum import Enum
from typing import Pattern

import mysql.connector
from mysql.connector import MySQLConnection
from mysql.connector.cursor import MySQLCursor
from pymysqlreplication import BinLogStreamReader
from pymysqlreplication.column import Column
from pymysqlreplication.constants import FIELD_TYPE
from pymysqlreplication.event import BinLogEvent, GtidEvent, HeartbeatLogEvent, QueryEvent, RotateEvent, RowsQueryLogEvent, XidEvent
from pymysqlreplication.row_event import DeleteRowsEvent, TableMapEvent, UpdateRowsEvent, WriteRowsEvent

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
class Logger:
    def __init__(self, module: str) -> None:
        self.module = module

    def __log(self, message: str, level: str, file):
        time_stamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        print(f"{time_stamp} -- {level} -- {self.module} -- {message}", file=file)

    def info(self, message: str):
        self.__log(message, "INFO", sys.stdout)

    def error(self, message: str):
        self.__log(message, "ERROR", sys.stderr)

    def warning(self, message: str):
        self.__log(message, "WARNING", sys.stdout)

    def debug(
        self,
        message: str,
    ):
        self.__log(message, "DEBUG", sys.stdout)


class NonDMLType(Enum):
    CREATEDATABASE = "nondml_createdatabase"
    DROPDATABASE = "nondml_dropdatabase"
    ALTERDATABASE = "nondml_alterdatabase"
    CREATETABLE = "nondml_createtable"
    ALTERTABLE = "nondml_altertable"
    DROPTABLE = "nondml_droptable"
    RENAMETABLE = "nondml_renametable"
    TRUNCATETABLE = "nondml_truncatetable"
    CREATEINDEX = "nondml_createindex"


@dataclass
class OperationNonDML:
    oper_type: NonDMLType
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


def generate_condition_and_values(primary_key: str | tuple, values: dict[str, any]):
    if primary_key:
        if isinstance(primary_key, str):
            primary_key = (primary_key,)
        condition = " AND ".join(f"`{k}` = %s" for k in primary_key)
        primary_values = tuple(values[k] for k in primary_key)
    else:
        condition = " AND ".join(f"`{k}` = %s" for k in values.keys())
        primary_values = tuple(values.values())

    return (condition, primary_values)


def dmlOperation2Sql(operation: OperationDML) -> tuple[str, tuple]:
    if isinstance(operation, OperationDMLInsert):
        keys = ", ".join(operation.values.keys())
        values_placeholder = ",".join(f'{"%s"}' for _ in operation.values)
        sql = f"REPLACE INTO `{operation.schema}`.`{operation.table}`({keys}) VALUES({values_placeholder})"
        params = tuple(operation.values.values())
    elif isinstance(operation, OperationDMLDelete):
        (condition, primary_values) = generate_condition_and_values(operation.primary_key, operation.values)
        sql = f"DELETE FROM `{operation.schema}`.`{operation.table}` WHERE {condition}"
        params = primary_values
    elif isinstance(operation, OperationDMLUpdate):
        set_clause = ", ".join([f"{k} = %s" for k in operation.after_values.keys()])
        (condition, primary_values) = generate_condition_and_values(operation.primary_key, operation.before_values)
        sql = f"UPDATE `{operation.schema}`.`{operation.table}` SET {set_clause} WHERE {condition}"
        params = tuple(operation.after_values.values()) + tuple(primary_values)

    return (sql, params)


nondml_patterns: list[tuple[Pattern[str], NonDMLType]] = [
    (
        re.compile(
            r"\s*create\s+database\s+(?:if\s+not\s+exists\s+)?`?(\w+)`?\s*",
            re.IGNORECASE,
        ),
        NonDMLType.CREATEDATABASE,
    ),
    (
        re.compile(
            r"\s*alter\s+database\s+`?(\w+)`?\s*",
            re.IGNORECASE,
        ),
        NonDMLType.ALTERDATABASE,
    ),
    (
        re.compile(
            r"\s*drop\s+database\s+(?:if\s+exists\s+)?`?(\w+)`?\s*",
            re.IGNORECASE,
        ),
        NonDMLType.DROPDATABASE,
    ),
    (
        re.compile(
            r"\s*create\s+table\s+(?:`?(\w+)`?\.)?`?(\w+)`?\s?.*",
            re.IGNORECASE,
        ),
        NonDMLType.CREATETABLE,
    ),
    (
        re.compile(
            r"\s*alter\s+table\s+(?:`?(\w+)`?\.)?`?(\w+)`?\s+.*",
            re.IGNORECASE,
        ),
        NonDMLType.ALTERTABLE,
    ),
    (
        re.compile(
            r"\s*drop\s+table\s+(?:`?(\w+)`?\.)?`?(\w+)`?\s*",
            re.IGNORECASE,
        ),
        NonDMLType.DROPTABLE,
    ),
    (
        re.compile(
            r"\s*rename\s+table\s+(?:`?(\w+)`?\.)?`?(\w+)`?\s+.*",
            re.IGNORECASE,
        ),
        NonDMLType.RENAMETABLE,
    ),
    (
        re.compile(
            r"\s*truncate\s+table\s+(?:`?(\w+)`?\.)?`?(\w+)`?\s*",
            re.IGNORECASE,
        ),
        NonDMLType.TRUNCATETABLE,
    ),
    (
        re.compile(
            r"\s*create\s+index\s+\w+\s+on\s+(?:`?(\w+)`?\.)?`?(\w+)`?\s?.*",
            re.IGNORECASE,
        ),
        NonDMLType.CREATEINDEX,
    ),
]


def extract_schema(statement: str) -> tuple[NonDMLType | None, str | None, str | None]:
    for regex, ddl_type in nondml_patterns:
        match = regex.search(statement)
        if match:
            groups = match.groups()
            if ddl_type in {NonDMLType.CREATEDATABASE, NonDMLType.ALTERDATABASE, NonDMLType.DROPDATABASE}:
                # return ddl_type, groups[0], None
                return (ddl_type, None, None)
            elif ddl_type in {NonDMLType.CREATETABLE, NonDMLType.ALTERTABLE, NonDMLType.DROPTABLE, NonDMLType.TRUNCATETABLE, NonDMLType.RENAMETABLE, NonDMLType.CREATEINDEX}:
                return (ddl_type, groups[0], groups[1])

    return (None, None, None)


class MysqlClient:
    def __init__(self, settings) -> None:
        self.logger = Logger("mysqlclient")

        self.con: MySQLConnection = mysql.connector.connect(**settings["connection_settings"])
        self.cur: MySQLCursor = self.con.cursor()
        self.con.autocommit = False
        self.dml_cnt = 0

    def push_begin(self):
        print("1111", self.con.in_transaction)
        if self.con.in_transaction:
            return
        self.con.start_transaction()
        print("1111", self.con.in_transaction)

    def push_dml(self, sql_text: str, params: tuple) -> None:
        self.cur.execute(sql_text, params)
        self.dml_cnt += 1

    def push_nondml(self, db: str | None, sql_text: str) -> None:
        if db:
            self.con.database = db
        with self.con.cursor() as cur:
            cur.execute(sql_text)

    def push_commit(self) -> None:
        if self.dml_cnt >= 1 and self.con.in_transaction:
            self.con.commit()
        else:
            self.con.rollback()
        self.dml_cnt = 0

    def push_rollback(self) -> None:
        self.con.rollback()


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
    def __init__(self, source_settings: dict) -> None:
        self.logger = Logger("mysqlreplication")

        self.binlogeventstream: Iterator[BinLogEvent] = BinLogStreamReader(**source_settings)

    def __handle_table_map_event(self, event: TableMapEvent):
        pass

    def __handle_event_update_rows(self, event: UpdateRowsEvent) -> list[OperationDML]:
        ls = []
        for row in event.rows:
            new_after_values = reset_values(row["after_values"], event.columns)
            new_before_values = reset_values(row["before_values"], event.columns)
            ls.append(OperationDMLUpdate(event.schema, event.table, new_after_values, new_before_values, event.primary_key))
        return ls

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
        (server_uuid, xid) = event.gtid.split(":")
        return OperationGtid(server_uuid, xid, event.last_committed)

    def __handle_event_rotate(self, event: RotateEvent):
        self.logfile = event.next_binlog

    def __handle_event_xid(self, event: XidEvent):
        return OperationCommit()

    def __handle_event_query(self, event: QueryEvent):
        if event.query == "BEGIN":
            return OperationBegin()
        elif event.query == "COMMIT":
            self.logger.warning('empty trx, query:"COMMIT"')
            return None
        else:
            (ddltype, db, tab) = extract_schema(event.query)

            if ddltype is None:
                raise Exception("not know queryevent query", event.query)

            if ddltype in (
                NonDMLType.CREATEDATABASE,
                NonDMLType.DROPDATABASE,
                NonDMLType.ALTERDATABASE,
            ):
                return OperationNonDML(ddltype, None, None, event.query)

            else:
                if ddltype in (
                    NonDMLType.CREATETABLE,
                    NonDMLType.ALTERTABLE,
                    NonDMLType.TRUNCATETABLE,
                    NonDMLType.CREATEINDEX,
                    NonDMLType.RENAMETABLE,
                    NonDMLType.DROPTABLE,
                ):
                    schema = event.schema.decode("utf-8") if event.schema_length >= 1 else db

                    if schema:
                        return OperationNonDML(ddltype, schema, tab, event.query)
                    else:
                        raise Exception("db not know", event.__dict__)

    def __handle_event_heartheatlog(self, event: HeartbeatLogEvent):
        return OperationHeartbeat()

    def operation_stream(
        self,
    ):
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
                    if isinstance(
                        operation,
                        list,
                    ):
                        for o in operation:
                            yield (
                                log_file,
                                log_pos,
                                binlogevent.timestamp,
                                o,
                            )
                    else:
                        yield (
                            log_file,
                            log_pos,
                            binlogevent.timestamp,
                            operation,
                        )
            log_pos = end_log_pos


def format_filter_tables(tables: list[str]) -> dict:
    filter_dict: dict[str, list[str]] = {}
    for dbtab in tables:
        (db, tab) = dbtab.split(".")
        if db in filter_dict:
            filter_dict[db].append(tab)
        else:
            filter_dict[db] = [tab]
    return filter_dict


class MysqlRowCompare:
    def __init__(self, mysql_source_connection_settings: dict, mysql_target_connection_settings: dict):
        pass

    def compare_key(self, dbtab: str, primarykey: dict[str, any]):
        pass

    def compare_nonkey(self, dbtab: str, values: list[any]):
        pass


class MetricAttr(Enum):
    DML_INSERT = "dml_insert"
    DML_UPDATE = "dml_update"
    DML_DELETE = "dml_delete"
    NONDML = "nondml"
    TRX = "trx"
    MERGE_TRX = "merge_trx"
    DELAY = "delay"


class MetricController:
    def __init__(self) -> None:
        self.__file = "metrics.json"
        self.__metrics = {attr.value: 0 for attr in list(MetricAttr) + list(NonDMLType)}

    def increment(self, attr: MetricAttr | NonDMLType) -> None:
        if attr.value in self.__metrics and attr.value != MetricAttr.DELAY.value:
            self.__metrics[attr.value] += 1

    def set_delay(self, value: int):
        self.__metrics["delay"] += value

    def persist(
        self,
    ):
        with open(self.__file, "w", encoding="utf8") as f:
            json.dump(asdict(self.__metrics), f, ensure_ascii=False)


class Checkpoint:
    def __init__(
        self,
        gtidset: str,
    ) -> None:
        self.checkpoint_gtidset: str = self.__get_checkpoint_gtidset(gtidset)

    def persist_checkpoint_gitdset(
        self,
        gtidset: dict[
            str,
            int,
        ],
    ):
        print("checkpoint", gtidset)
        with open("ckpt.json", "w", encoding="utf8") as f:
            return json.dump(gtidset, f)

    def __format_gtidset(self, gtidsets: str) -> dict[str, int]:
        _gtidset = {}
        for gtidset in gtidsets.split(","):
            gtidsetf = gtidset.split(":")
            _gtidset[gtidsetf[0]] = int(gtidsetf[-1].split("-")[-1] if "-" in gtidsetf[-1] else gtidsetf[-1])
        return _gtidset

    def __get_checkpoint_gtidset(self, gtidset: str) -> str:
        _start_gtidset = self.__format_gtidset(gtidset)
        if os.path.exists("ckpt.json"):
            with open("ckpt.json", "r", encoding="utf8") as f:
                _gtidset: dict[str, int] = json.load(f)
                for (
                    _server_uuid,
                    _xid,
                ) in _gtidset.items():
                    if _server_uuid in _start_gtidset:
                        _start_gtidset[_server_uuid] = max(
                            _start_gtidset[_server_uuid],
                            _xid,
                        )
                    else:
                        _start_gtidset[_server_uuid] = _xid

        return ", ".join(f"{key}:1-{value}" for key, value in _start_gtidset.items())


class MysqlSync:
    def __init__(self, mysql_source_settings: dict, mysql_target_settings: dict, mysql_sync_settings: dict) -> None:
        self.logger = Logger("mysqlsync")
        self.metric = MetricController()

        self.args_mysql_sync_merge_trx = mysql_sync_settings["merge_trx"]

        self.ckpt = Checkpoint(mysql_source_settings["auto_position"])

        self.logger.info(f"start gtidset {self.ckpt.checkpoint_gtidset}")
        if self.ckpt:
            mysql_source_settings["auto_position"] = self.ckpt.checkpoint_gtidset

        self.mr = MysqlReplication(mysql_source_settings)
        self.mc = MysqlClient(mysql_target_settings)

        self.last_committed = -1
        self.allow_commit = False
        self.last_operation = None

        self.allow_ckeckpoint = False
        self.checkpoint_gtidset = self.ckpt.checkpoint_gtidset
        self.checkpoint_position = ()

        self.latest_gtidset = {}
        self.latest_position = ()

        self.exclude_tables: dict[str, list[str]] = {}

    def __merge_commit(
        self,
    ):
        if self.allow_commit:
            self.mc.push_commit()
            self.allow_commit = False
            self.metric.increment(MetricAttr.MERGE_TRX)
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
                    self.metric.increment(operation.oper_type)
                except Exception as e:
                    self.logger.error(f"error push_nondml {operation.schema} {operation.query} {e}")
                    sys.exit(1)
            elif isinstance(
                operation,
                OperationDML,
            ):
                if operation.schema in self.exclude_tables and operation.table in self.exclude_tables[operation.schema]:
                    self.logger.info(f"exclude tables `{operation.schema}`.`{operation.table}`")
                    continue

                if type(operation) == OperationDMLDelete:
                    self.metric.increment(MetricAttr.DML_DELETE)

                elif type(operation) == OperationDMLInsert:
                    self.metric.increment(MetricAttr.DML_INSERT)
                elif type(operation) == OperationDMLUpdate:
                    self.metric.increment(MetricAttr.DML_UPDATE)

                try:
                    (sql_text, params) = dmlOperation2Sql(operation)
                    self.mc.push_dml(sql_text, params)
                except Exception as e:
                    self.logger.error(f" dml error, {operation}, {e}")
            elif type(operation) == OperationBegin:
                self.mc.push_begin()
            elif type(operation) == OperationCommit:
                self.metric.increment(MetricAttr.TRX)

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
                self.latest_position = (
                    log_file,
                    log_pos,
                )
            elif type(operation) == OperationHeartbeat and type(self.last_operation) == OperationHeartbeat:
                _timestamp = time.time()
                self.__merge_commit()
            else:
                pass

            if type(operation) != OperationHeartbeat:
                _timestamp = timestamp

            self.metric.set_delay(int(time.time() - _timestamp))

            _ts_1 = time.time()
            if _ts_1 >= _ts_0 + 10:
                self.metric.persist()

                _ts_0 = _ts_1

                if self.allow_commit and not self.allow_ckeckpoint:
                    self.allow_ckeckpoint = True

            self.last_operation = operation

        self.__merge_commit()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--config", required=True, help="Path to the configuration file")
    args = parser.parse_args()
    if args.config:
        spec = importlib.util.spec_from_file_location("settings", args.config)
        settings = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(settings)

        print(f"source settings: {settings.source_settings}")
        print(f"target settings: {settings.target_settings}")
        print(f"sync settings: {settings.sync_settings}")
        MysqlSync(settings.source_settings, settings.target_settings, settings.sync_settings).run()


if __name__ == "__main__":
    main()
