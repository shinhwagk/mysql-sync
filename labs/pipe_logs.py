import subprocess
import datetime
import threading

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

def run_pipeline():
    mysqlbinlog_cmd = ['mysqlbinlog', 'mysql-bin.000001']
    mysqlbinlog_statistics_cmd = ['mysqlbinlog_statistics']
    mysql_cmd = ['mysql', '-u', 'username', '-p', 'password', '-h', 'hostname', 'database_name']
    
    p1 = subprocess.Popen(mysqlbinlog_cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    p2 = subprocess.Popen(mysqlbinlog_statistics_cmd, stdin=p1.stdout, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    p3 = subprocess.Popen(mysql_cmd, stdin=p2.stdout, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

    # 创建线程来处理每个进程的stderr输出
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

if __name__ == '__main__':
    run_pipeline()
