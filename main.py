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


def kill_p1(p1: subprocess.Popen):
    def signal_handler(signum: int, frame) -> NoReturn:
        print("Received SIGTERM, shutting down...")
        kill = False
        while True:
            print(1111, p1, p1.poll())
            if p1 is not None and p1.poll() is None and kill == False:
                print("Received SIGTERM, shutting down...1")
                p1.terminate()
                print("kill1")
                p1.wait()
                print("kill1222")

                break
            kill = True
            time.sleep(1)
        # sys.exit(0)

    return signal_handler


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
) -> list[str]:
    return [
        "mysqlbinlog",
        f"--host={host}",
        f"--port={port}",
        f"--user={user}",
        f"--password={password}",
        "--read-from-remote-source=BINLOG-DUMP-GTIDS",
        "--compression-algorithms=zstd",
        "--zstd-compression-level=3",
        "--verify-binlog-checksum",
        # "--to-last-log",
        "--stop-never",
        f"--connection-server-id={server_id}",
        "--verbose",
        "--verbose",  # 重复的 '--verbose' 表示更详细的输出
        "--idempotent",
        "--force-read",
        "--print-table-metadata",
        start_binlogfile,
    ]


def make_cmd_cmd2() -> list[str]:
    return ["mysqlbinlog-statistics"]


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


def log_writer(log_pipe: Optional[IO[bytes]], prefix: str) -> None:
    current_day = None
    log_file: Optional[IO[str]] = None
    try:
        if log_pipe is None:
            raise ValueError("Log pipe cannot be None")

        while True:
            line = log_pipe.readline()
            if not line:
                print("log none over.")
                break
            today = datetime.datetime.now().strftime("%Y-%m-%d")
            if today != current_day:
                if log_file:
                    log_file.close()
                log_file = open(f"{prefix}_errors_{today}.log", "a")
                current_day = today

            log_file.write(line.decode("utf-8"))
            log_file.flush()
    finally:
        if log_file:
            log_file.close()


def binlogReplicationWatcher(host, user, password):
    try:
        with mysql.connector.connect(host=host, user=user, password=password) as conn:
            with conn.cursor() as cursor:
                cursor.execute("CREATE DATABASE IF NOT EXISTS mysqlbinlogsync")
                cursor.execute("USE mysqlbinlogsync")
                cursor.execute(
                    "CREATE TABLE IF NOT EXISTS sync_table (id INT PRIMARY KEY)"
                )

                while True:
                    cursor.execute("INSERT INTO sync_table VALUES (1)")
                    conn.commit()
                    time.sleep(0.1)

                    cursor.execute("DELETE FROM sync_table WHERE id = 1")
                    conn.commit()
                    time.sleep(0.1)

    except mysql.connector.Error as e:
        print(f"Error: {e}")


def parse_connection_string(conn_str: str) -> dict:
    user_info, host_info = conn_str.split("@")
    username, password = user_info.split("/")
    host, port = host_info.split(":")
    return {"user": username, "password": password, "host": host, "port": port}


def parse_args():
    parser = argparse.ArgumentParser(description="Process some integers.")

    parser.add_argument(
        "--source_dsn", type=str, required=True, help="The DSN for the source database"
    )

    parser.add_argument(
        "--target_dsn", type=str, required=True, help="The DSN for the target database"
    )

    parser.add_argument(
        "--start_gtid",
        type=str,
        required=False,
        help="The starting GTID for the operations",
    )

    args = parser.parse_args()


def kill_p1():
    global p1

    def signal_handler(sig, frame):
        if p1.poll() is None:
            p1.terminate()
            p1.wait()
            print("子进程已终止。")

    return signal_handler


def run_pipeline(
    mysqlbinlog_cmd: list[str],
    mysqlbinlog_statistics_cmd: list[str],
    mysql_cmd: list[str],
):
    global p1
    # mysqlbinlog_cmd = ["mysqlbinlog", "mysql-bin.000001"]
    # mysqlbinlog_statistics_cmd = ["mysqlbinlog_statistics"]
    # mysql_cmd = [
    #     "mysql",
    #     "-u",
    #     "username",
    #     "-p",
    #     "password",
    #     "-h",
    #     "hostname",
    #     "database_name",
    # ]
    signal_handler = kill_p1()
    signal.signal(signal.SIGTERM, signal_handler)
    signal.signal(signal.SIGINT, signal_handler)

    p1 = subprocess.Popen(
        mysqlbinlog_cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE
    )

    # print(1, p1, p1.poll())
    # # p1.terminate()
    # handler = kill_p1(p1)

    # signal.signal(signal.SIGTERM, handler)

    # time.sleep(15)

    # p2 = subprocess.Popen(
    #     mysqlbinlog_statistics_cmd,
    #     stdin=p1.stdout,
    #     stdout=subprocess.PIPE,
    #     stderr=subprocess.PIPE,
    # )
    # p3 = subprocess.Popen(
    #     mysql_cmd, stdin=p2.stdout, stdout=subprocess.PIPE, stderr=subprocess.PIPE
    # )

    thread_p1 = threading.Thread(target=log_writer, args=(p1.stdout, "p1"))
    # thread_p2 = threading.Thread(target=log_writer, args=(p2.stderr, "p2"))
    # thread_p3 = threading.Thread(target=log_writer, args=(p3.stderr, "p3"))

    print("end1")
    thread_p1.start()

    # print("sleep2")
    # time.sleep(5)
    # print("sleep1")
    # p1.terminate()
    # thread_p2.start()
    # thread_p3.start()

    print("end2")
    p1.communicate()
    print("end3")
    # p2.wait()
    # p3.wait()

    thread_p1.join()
    print("end4")
    # thread_p2.join()
    # thread_p3.join()


def main():
    global p1
    # handler =

    parse_args()
    if is_redhat_family():
        download_mysql_client("8.0.36")

        s_dsn = parse_connection_string("root/example@db1:3306")
        t_dsn = parse_connection_string("root/example@db2:3306")
        cmd1 = make_cmd_cmd1(
            **s_dsn, server_id=111, start_binlogfile="mysql-bin.000001"
        )
        cmd2 = make_cmd_cmd2()
        cmd3 = make_cmd_cmd3(**t_dsn)
        run_pipeline(cmd1, cmd2, cmd3)
    # run_pipeline()


if __name__ == "__main__":
    main()
