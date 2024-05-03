import argparse
import datetime
import json
import os
import signal
import subprocess
import sys
import threading
import time
from typing import IO, BinaryIO, NoReturn, Optional, TextIO

import mysql.connector
from mysql.connector import Error

parser = argparse.ArgumentParser(description="Process some integers.")
parser.add_argument("--source-dsn", type=str, required=True, help="The DSN for the source database")
parser.add_argument("--target-dsn", type=str, required=True, help="The DSN for the target database")
parser.add_argument("--mysqlbinlog-statistics", type=str, required=False, help="The DSN for the target database")
parser.add_argument("--mysqlbinlog-zstd-compression-level", type=int, required=False, help="The DSN for the target database")
parser.add_argument("--mysqlbinlog-connection-server-id", type=int, required=False, default=99999, help="The DSN for the target database")
parser.add_argument("--mysqlbinlog-exclude-gtids", type=str, required=False, help="The starting GTID for the operations")
parser.add_argument("--mysqlbinlog-stop-never", action="store_true", help="The starting GTID for the operations")

args = parser.parse_args()
# def is_redhat_family():
#     os_release_file = "/etc/os-release"

#     if os.path.exists(os_release_file):
#         with open(os_release_file, "r") as file:
#             for line in file:
#                 key, value = line.strip().replace('"', "").split("=", 1)
#                 if key == "ID" and value in {"rhel", "centos", "fedora", "rocky"}:
#                     return True
#     return False


# def mkdirs():
#     for d in ["bin", "logs"]:
#         os.mkdirs(d)


# def download_mysqlbinlog_statistics(version: str):
#     pass


# def download_mysql_client(version: str):
#     # must redhat family
#     try:
#         for cmd in [
#             [
#                 "dnf",
#                 "install",
#                 "-y",
#                 "https://dev.mysql.com/get/mysql80-community-release-el9-5.noarch.rpm",
#             ],
#             ["dnf", "install", "-y", f"mysql-community-client-{version}"],
#         ]:
#             subprocess.run(cmd, check=True, text=True, capture_output=True)
#         print("mysqlbinlog & mysql installed.")
#     except subprocess.CalledProcessError as e:
#         print(e.stderr)


def compare_gtid():
    pass


def make_cmd_cmd1(
    host: str,
    port: str,
    user: str,
    password: str,
    server_id: str,
    start_binlogfile: str,
    compression_level: Optional[str],
    exclude_gtids: Optional[str] = None,
    stop_never: bool = False,
) -> list[str]:
    command = [
        "mysqlbinlog",
        f"--host={host}",
        f"--port={port}",
        f"--user={user}",
        f"--password={password}",
        "--read-from-remote-source=BINLOG-DUMP-GTIDS",
        "--verify-binlog-checksum",
        f"--connection-server-id={server_id}",
        "--verbose",
        "--verbose",
        "--idempotent",
        "--force-read",
        "--print-table-metadata",
    ]
    command += [f"--compression-algorithms=zstd", f"--zstd-compression-level={compression_level}"] if compression_level else []
    command += [f"--exclude-gtids={exclude_gtids}"] if exclude_gtids else []
    command.append("--stop-never" if stop_never else "--to-last-log")
    command.append(start_binlogfile)

    return command


def make_cmd_cmd2() -> list[str]:
    # return ["mysqlbinlog-statistics"]
    return ["./target/debug/mysqlbinlog-statistics"]


def make_cmd_cmd3(host: str, port: str, user: str, password: str) -> list[str]:
    return [
        "mysql",
        f"--host={host}",
        f"--port={port}",
        f"--user={user}",
        f"--password={password}",
        "--verbose",
        "--verbose",
        "--verbose",
    ]


def log_writer(log_pipe: IO[bytes], prefix: str) -> NoReturn:
    current_day = None
    log_file: Optional[IO[str]] = None
    cnt = 1
    try:
        while True:
            line = log_pipe.readline()
            if not line:
                print(f"{prefix} log none over.")
                break
            today = datetime.datetime.now().strftime("%Y-%m-%d")
            if today != current_day:
                if log_file:
                    log_file.close()
                log_file = open(f"{prefix}.{today}.log", "a")
                current_day = today

            if prefix == "mysql.stdout":
                cnt += 1
                if cnt % 1000000 == 0:
                    print("mysql.stdout", cnt)

            log_file.write(line.decode("utf-8"))
            log_file.flush()
    finally:
        if log_file:
            log_file.close()


def binlogReplicationWatcher(con: mysql.connector.MySQLConnection, stop_event: threading.Event):
    try:
        with con.cursor() as cur:
            cur.execute("DROP DATABASE IF EXISTS mysqlbinlogsync")
            cur.execute("CREATE DATABASE IF NOT EXISTS mysqlbinlogsync")
            cur.execute("CREATE TABLE IF NOT EXISTS mysqlbinlogsync.sync_table (id INT PRIMARY KEY, ts DATETIME)")

            while not stop_event.is_set():
                cur.execute("REPLACE INTO mysqlbinlogsync.sync_table VALUES(1, now())")
                con.commit()
                time.sleep(0.2)
    except mysql.connector.Error as e:
        print(f"Error: {e}")


def parse_connection_string(conn_str: str) -> dict:
    user_info, host_info = conn_str.split("@")
    username, password = user_info.split("/")
    host, port = host_info.split(":")
    return {"user": username, "password": password, "host": host, "port": port}


def query_first_binlogfile(con: mysql.connector.MySQLConnection) -> str:
    with con.cursor(buffered=True) as cur:
        cur.execute("SHOW BINARY LOGS")
        return cur.fetchone()[0]


def query_gtid_set(con: mysql.connector.MySQLConnection) -> str:
    with con.cursor(buffered=True) as cur:
        cur.execute("show master status")
        return cur.fetchone()[4]


def query_server_uuid(con: mysql.connector.MySQLConnection) -> str:
    with con.cursor(buffered=True) as cur:
        cur.execute("select @@server_uuid")
        return cur.fetchone()[0]


def query_server_id(con: mysql.connector.MySQLConnection) -> str:
    with con.cursor(buffered=True) as cur:
        cur.execute("select @@server_id")
        return cur.fetchone()[0]


def ext_gtid():
    pass


def main():
    s_dsn = parse_connection_string(args.source_dsn)
    t_dsn = parse_connection_string(args.target_dsn)

    s_conn = mysql.connector.connect(**s_dsn)
    t_conn = mysql.connector.connect(**t_dsn)

    binlogfile = query_first_binlogfile(s_conn)
    gtid_set = query_gtid_set(t_conn)

    server_uuid = query_server_uuid(s_conn)
    server_id = query_server_id(s_conn)

    mysqlbinlog_connection_server_id = args.mysqlbinlog_connection_server_id

    exclude_gtids= gtid_set if len(gtid_set) >= 1 else None

    mysqlbinlog_cmd = make_cmd_cmd1(
        **s_dsn,
        server_id=mysqlbinlog_connection_server_id,
        start_binlogfile=binlogfile,
        exclude_gtids=exclude_gtids,
        compression_level=None,
        stop_never=args.mysqlbinlog_stop_never,
    )
    print("cmd1", " ".join(mysqlbinlog_cmd))
    mysqlbinlog_statistics_cmd = make_cmd_cmd2()
    print("cmd2", " ".join(mysqlbinlog_statistics_cmd))
    mysql_cmd = make_cmd_cmd3(**t_dsn)
    print("cmd3", " ".join(mysql_cmd))

    # if args.mysqlbinlog_stop_never:
    #     se = threading.Event()
    #     tx = threading.Thread(target=binlogReplicationWatcher, args=(s_conn, se))
    #     tx.start()

    p1 = subprocess.Popen(mysqlbinlog_cmd, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL)
    p2 = subprocess.Popen(mysqlbinlog_statistics_cmd, stdin=p1.stdout, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL)
    p3 = subprocess.Popen(mysql_cmd, stdin=p2.stdout, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    p1.stdout.close()
    p2.stdout.close()

    # p1.stderr.close()
    # p2.stderr.close()
    # p3.stderr.close()
    # p3.stdout.close()

    def handler(sig, frame):
        # se.set()
        if p1.poll() is None:
            p1.terminate()

    signal.signal(signal.SIGTERM, handler)
    signal.signal(signal.SIGINT, handler)

    threads: list[threading.Thread] = []

    # for proc, name in [
    #     (p1.stderr, "mysqlbinlog.stderr"),
    #     (p2.stderr, "mysqlbinlog-statistics.stderr"),
    #     (p3.stderr, "mysql.stderr"),
    #     (p3.stdout, "mysql.stdout"),
    # ]:
    #     t = threading.Thread(target=log_writer, args=(proc, name))
    #     t.start()
    #     threads.append(t)

    print("process 1wait success")

    for p in [p1, p2, p3]:
        p.wait()
        print(f"p shifang {p}")

    print("process wait success")

    for t in threads:
        t.join()

    # if args.mysqlbinlog_stop_never:
    #     se.set()
    #     tx.join()

    print("logger wait success")

    print("stder wait success")


if __name__ == "__main__":
    main()
