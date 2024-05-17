use std::io::{self, BufRead};
use std::sync::mpsc::{self, TryRecvError};
use std::thread;
use std::time::{Duration, Instant};

fn main() {
    let (tx, rx) = mpsc::channel();
    let timeout = Duration::from_secs(5); // 设置超时时间为5秒

    // 数据读取线程
    thread::spawn(move || {
        let stdin = io::stdin();
        let handle = stdin.lock();
        for line in handle.lines() {
            let line = line.expect("Failed to read line");
            tx.send(line).expect("Failed to send line");
        }
    });

    let mut last_time = Instant::now();

    loop {
        match rx.try_recv() {
            Ok(line) => {
                // 成功接收到数据，重置超时计时
                println!("Received: {}", line);
                last_time = Instant::now();
            }
            Err(TryRecvError::Empty) => {
                // 没有新数据，检查是否超时
                if last_time.elapsed() > timeout {
                    println!("Timeout, no new data received.");
                    break; // 超时结束循环
                }
                // 短暂休眠以减少CPU占用
                thread::sleep(Duration::from_millis(100));
            }
            Err(TryRecvError::Disconnected) => {
                // 数据源断开连接
                println!("Channel disconnected.");
                break;
            }
        }
    }
}
