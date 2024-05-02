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


def is_redhat_family():
    os_release_file = "/etc/os-release"

    if os.path.exists(os_release_file):
        with open(os_release_file, "r") as file:
            for line in file:
                key, value = line.strip().replace('"', "").split("=", 1)
                if key == "ID" and value in {"rhel", "centos", "fedora", "rocky"}:
                    return True
    return False


def mkdirs():
    for d in ["bin", "logs"]:
        os.mkdirs(d)


def download_mysqlbinlog_statistics(version: str):
    pass


def download_mysql_client(version: str):
    # must redhat family
    try:
        for cmd in [
            [
                "dnf",
                "install",
                "-y",
                "https://dev.mysql.com/get/mysql80-community-release-el9-5.noarch.rpm",
            ],
            ["dnf", "install", "-y", f"mysql-community-client-{version}"],
        ]:
            subprocess.run(cmd, check=True, text=True, capture_output=True)
        print("mysqlbinlog & mysql installed.")
    except subprocess.CalledProcessError as e:
        print(e.stderr)


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
    gtid: Optional[str] = None,
    stop_never: bool = False,
) -> list[str]:
    return (
        [
            "mysqlbinlog",
            f"--host={host}",
            f"--port={port}",
            f"--user={user}",
            f"--password={password}",
            "--read-from-remote-source=BINLOG-DUMP-GTIDS",
            "--verify-binlog-checksum",
        ]
        + [("--stop-never" if stop_never else "--to-last-log")]
        + [
            "--stop-never",
            f"--connection-server-id={server_id}",
            "--verbose",
            "--verbose",
            "--idempotent",
            "--force-read",
            "--print-table-metadata",
        ]
        + (["--compression-algorithms=zstd", f"--zstd-compression-level={compression_level}"] if compression_level else [])
        + ([f"--exclude-gtids={gtid}"] if gtid else [])
        + [start_binlogfile]
    )


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


def log_writer(log_pipe: IO[bytes], prefix: str) -> None:
    current_day = None
    log_file: Optional[IO[str]] = None
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

            log_file.write(line.decode("utf-8"))
            log_file.flush()
    finally:
        if log_file:
            log_file.close()


def binlogReplicationWatcher(con: mysql.connector.MySQLConnection):
    try:
        with con.cursor() as cur:
            cur.execute("CREATE DATABASE IF NOT EXISTS mysqlbinlogsync")
            cur.execute("USE mysqlbinlogsync")
            cur.execute("CREATE TABLE IF NOT EXISTS sync_table (id INT PRIMARY KEY)")

            while True:
                cur.execute("DELETE FROM sync_table")
                con.commit()
                time.sleep(0.2)

                cur.execute("INSERT INTO sync_table VALUES (1)")
                con.commit()
                time.sleep(0.2)
    except mysql.connector.Error as e:
        print(f"Error: {e}")


def parse_connection_string(conn_str: str) -> dict:
    user_info, host_info = conn_str.split("@")
    username, password = user_info.split("/")
    host, port = host_info.split(":")
    return {"user": username, "password": password, "host": host, "port": port}


def parse_args():
    parser = argparse.ArgumentParser(description="Process some integers.")
    parser.add_argument("--source-dsn", type=str, required=True, help="The DSN for the source database")
    parser.add_argument("--target-dsn", type=str, required=True, help="The DSN for the target database")
    parser.add_argument("--mysqlbinlog-zstd-compression-level", type=int, required=False, help="The DSN for the target database")
    parser.add_argument("--mysqlbinlog-connection-server-id", type=int, required=False, help="The DSN for the target database")
    parser.add_argument("--mysqlbinlog-exclude-gtids", type=str, required=False, help="The starting GTID for the operations")
    parser.add_argument("--mysqlbinlog-stop-never", type=bool, required=False, default=False, help="The starting GTID for the operations")

    return parser.parse_args()


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


def run_pipeline(mysqlbinlog_cmd: list[str], mysqlbinlog_statistics_cmd: list[str], mysql_cmd: list[str]) -> NoReturn:
    p1 = subprocess.Popen(mysqlbinlog_cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

    p2 = subprocess.Popen(mysqlbinlog_statistics_cmd, stdin=p1.stdout, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    p3 = subprocess.Popen(mysql_cmd, stdin=p2.stdout, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    p1.stdout.close()
    p2.stdout.close()

    def handler(sig, frame):
        if p1.poll() is None:
            p1.terminate()

    signal.signal(signal.SIGTERM, handler)
    signal.signal(signal.SIGINT, handler)

    threads: list[threading.Thread] = []

    for proc, name in [
        (p1.stderr, "mysqlbinlog.stderr"),
        (p2.stderr, "mysqlbinlog-statistics.stderr"),
        (p3.stderr, "mysql.stderr"),
        (p3.stdout, "mysql.stdout"),
    ]:
        t = threading.Thread(target=log_writer, args=(proc, name))
        t.start()
        threads.append(t)

    print("process 1wait success")

    for p in [p1, p2, p3]:
        p.wait()
        print(f"p shifang {p}")

    print("process wait success")

    for t in threads:
        t.join()

    print("logger wait success")

    # for p in [p1.stderr, p2.stderr, p3.stderr, p3.stdout]:
    #     p.close()

    print("stder wait success")


def ext_gtid():
    pass


def main():
    args = parse_args()

    if is_redhat_family():
        download_mysql_client("8.0.36")

    s_dsn = parse_connection_string(args.source_dsn)
    t_dsn = parse_connection_string(args.target_dsn)

    s_conn = mysql.connector.connect(**s_dsn)
    t_conn = mysql.connector.connect(**t_dsn)

    binlogfile = query_first_binlogfile(s_conn)
    gtid_set = query_gtid_set(t_conn)

    server_uuid = query_server_uuid(s_conn)

    print("gtid_set", gtid_set)
    gtida = None
    if len(gtid_set) >= 1:
        for gtid in gtid_set.split(","):
            if gtid.startswith(server_uuid):
                gtida = gtid
    cmd1 = make_cmd_cmd1(
        **s_dsn, server_id=111, start_binlogfile=binlogfile, gtid=gtida, compression_level=None, stop_never=args.mysqlbinlog_stop_never
    )
    print("cmd1", " ".join(cmd1))
    cmd2 = make_cmd_cmd2()
    print("cmd2", " ".join(cmd2))
    cmd3 = make_cmd_cmd3(**t_dsn)
    print("cmd3", " ".join(cmd3))

    t = threading.Thread(target=binlogReplicationWatcher, args=(s_conn,))
    t.start()

    run_pipeline(cmd1, cmd2, cmd3)
    # run_pipeline()


if __name__ == "__main__":
    main()
