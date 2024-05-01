import subprocess
import datetime
import threading
import os
import subprocess
import threading
import signal
import os
import sys
from subprocess import Popen
from typing import NoReturn
import argparse


def signal_handler(signum: int, frame) -> NoReturn:
    print("Received SIGTERM, shutting down...")
    global p1
    if p1 is not None and p1.poll() is None:
        p1.terminate()
        p1.wait()
    sys.exit(0)

def mkdirs ():
    for d in ["bin","logs"]:
        os.mkdirs(d)

def download_mysqlbinlog_statistics(version:str):
    pass

def download_mysql_client(version:str):
    # must redhat family
    for cmd in  [['dnf', 'install', '-y', 'https://dev.mysql.com/get/mysql80-community-release-el9-5.noarch.rpm'],['dnf', 'install', '-y', 'mysql-community-client']]:
        try:
            result = subprocess.run(commcmdand, check=True, text=True, capture_output=True)
            print(result.stdout)
        except subprocess.CalledProcessError as e:
            print(e.stderr)


def compare_gtid():
    pass

def make_cmd_cmd1(host:str,port:str,user:str,password:str,server_id:str,start_binlogfile:str)->list[str]:
   return [
    'mysqlbinlog',
    f'--host={host}',
    f'--port={port}',
    f'--user={user}',
    f'--password={password}',
    '--read-from-remote-source=BINLOG-DUMP-GTIDS',
    '--compression-algorithms=zstd',
    '--zstd-compression-level=3',
    '--verify-binlog-checksum',
    '--to-last-log',
    f'--connection-server-id={server_id}',
    '--verbose',
    '--verbose',  # 重复的 '--verbose' 表示更详细的输出
    '--idempotent',
    '--force-read',
    '--print-table-metadata',
    start_binlogfile
]


def make_cmd_cmd2()->list[str]:
    return ["mysqlbinlog-statistics"]

def make_cmd_cmd3(host:str,port:str,user:str,password:str)->list[str]:
    return ["mysql",f'--host={host}',
    f'--port={port}',
    f'--user={user}',
    f'--password={password}',"--verbose"]

def log_writer(log_pipe, prefix):
    """ 管理日志输出，根据日期切换文件 """
    current_day = None
    log_file = None
    try:
        while True:
            line = log_pipe.readline()
            if not line:
                break
            today = datetime.datetime.now().strftime('%Y-%m-%d')
            if today != current_day:
                if log_file:
                    log_file.close()
                log_file = open(f"{prefix}_errors_{today}.log", 'a')
                current_day = today
            log_file.write(line.decode())
            log_file.flush()
    finally:
        if log_file:
            log_file.close()

import mysql.connector
from mysql.connector import Error
import time

def binlogReplicationWatcher(host, user, password):
    try:
        with mysql.connector.connect(host=host, user=user, password=password) as conn:
            with conn.cursor() as cursor:
                cursor.execute("CREATE DATABASE IF NOT EXISTS mysqlbinlogsync")
                cursor.execute("USE mysqlbinlogsync")
                cursor.execute("CREATE TABLE IF NOT EXISTS sync_table (id INT PRIMARY KEY)")

                while True:
                    cursor.execute("INSERT INTO sync_table VALUES (1)")
                    conn.commit()
                    time.sleep(0.1)
                    
                    cursor.execute("DELETE FROM sync_table WHERE id = 1")
                    conn.commit()
                    time.sleep(0.1)

    except Error as e:
        print(f"Error: {e}")

 
def signal_handler(signum, frame):
    print("Received termination signal, shutting down...")
    # 在这里添加任何需要的清理代码
    sys.exit(0)




def main(source_dsn, target_dsn, start_gtid):
    print(f"Source DSN: {source_dsn}")
    print(f"Target DSN: {target_dsn}")
    print(f"Start GTID: {start_gtid}")

import json

def parse_connection_string(conn_str)->dict:
    user_info, host_info = conn_str.split('@')
    username, password = user_info.split('/')
    host, port = host_info.split(':')
    return {
        "user": username,
        "password": password,
        "host": host,
        "port": port
    }




def parse_args():
    # 创建解析器
    parser = argparse.ArgumentParser(description="Process some integers.")

    # 添加 --source_dsn 参数
    parser.add_argument('--source_dsn', type=str, required=True,
                        help='The DSN for the source database')

    # 添加 --target_dsn 参数
    parser.add_argument('--target_dsn', type=str, required=True,
                        help='The DSN for the target database')

    # 添加 --start_gtid 参数
    parser.add_argument('--start_gtid', type=str, required=False,
                        help='The starting GTID for the operations')

    # 解析命令行参数
    args = parser.parse_args()

    # 将解析得到的参数传递给主函数
    main(args.source_dsn, args.target_dsn, args.start_gtid)

def run_pipeline():
    global p1

    mysqlbinlog_cmd = ['mysqlbinlog', 'mysql-bin.000001']
    mysqlbinlog_statistics_cmd = ['mysqlbinlog_statistics']
    mysql_cmd = ['mysql', '-u', 'username', '-p', 'password', '-h', 'hostname', 'database_name']
    
    p1 = subprocess.Popen(mysqlbinlog_cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    p2 = subprocess.Popen(mysqlbinlog_statistics_cmd, stdin=p1.stdout, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    p3 = subprocess.Popen(mysql_cmd, stdin=p2.stdout, stdout=subprocess.PIPE, stderr=subprocess.PIPE)


    thread_p1 = threading.Thread(target=log_writer, args=(p1.stderr, 'p1'))
    thread_p2 = threading.Thread(target=log_writer, args=(p2.stderr, 'p2'))
    thread_p3 = threading.Thread(target=log_writer, args=(p3.stderr, 'p3'))

    thread_p1.start()
    thread_p2.start()
    thread_p3.start()

    # 等待进程结束
    p1.wait()
    p2.wait()
    p3.wait()

    # 等待所有日志处理线程完成
    thread_p1.join()
    thread_p2.join()
    thread_p3.join()

def main():
    signal.signal(signal.SIGTERM, signal_handler)
    run_pipeline()

if __name__ == '__main__':
    main()
    
