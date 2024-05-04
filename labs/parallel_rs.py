import concurrent.futures
import itertools
import sys
import threading

class ReplicationHandler:
    def __init__(self, id):
        self.id = id
        self.last_committed_locks = {}  # 用于存储每个 last_committed 对应的锁
        self.last_sequence_number = {}  # 用于存储每个 last_committed 对应的上一个 sequence_number

    def handle_replication(self, last_committed, sequence_number):
        # 获取相应 last_committed 对应的锁
        lock = self.last_committed_locks.setdefault(last_committed, threading.Lock())

        # 获取上一个 sequence_number 对应的锁
        if sequence_number > 1:
            prev_sequence_number = self.last_sequence_number.setdefault(last_committed, 1)
            prev_lock = self.last_committed_locks.setdefault(last_committed + '_' + str(prev_sequence_number), threading.Lock())
            prev_lock.acquire()

        # 模拟 MySQL 并行复制的逻辑
        print(f"Handler {self.id}: Handling replication for last_committed={last_committed}, sequence_number={sequence_number}")

        # 更新上一个 sequence_number
        self.last_sequence_number[last_committed] = sequence_number

        # 释放当前 sequence_number 对应的锁
        lock.release()

def main():
    replication_handlers = [ReplicationHandler(i) for i in range(5)]  # 创建 5 个 ReplicationHandler 对象

    # 使用 ThreadPoolExecutor 创建线程池，最多同时运行 5 个线程
    with concurrent.futures.ThreadPoolExecutor(max_workers=5) as executor:
        while True:
            # 从标准输入读取数据
            data = (line.strip().split() for line in sys.stdin)

            # 使用 itertools.groupby 按照 last_committed 进行分组
            for last_committed, group in itertools.groupby(data, key=lambda x: x[10].split('=')[1]):
                # 创建一个任务队列
                queue = list(group)

                # 等待上一个任务完成后再提交下一个任务
                for i, entry in enumerate(queue):
                    sequence_number = int(entry[11].split('=')[1])
                    handler = replication_handlers[int(last_committed) % 5]  # 使用 last_committed 的值来选择处理对象
                    executor.submit(handler.handle_replication, last_committed, sequence_number)

                    if i > 0:
                        prev_sequence_number = int(queue[i - 1][11].split('=')[1])
                        if prev_sequence_number != sequence_number:  # 如果上一个任务的 sequence_number 与当前任务的 sequence_number 不同，则释放锁
                            prev_lock = handler.last_committed_locks.setdefault(last_committed + '_' + str(prev_sequence_number), threading.Lock())
                            prev_lock.release()

if __name__ == "__main__":
    main()
