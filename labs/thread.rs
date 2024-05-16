use std::sync::mpsc;
use std::thread;
use std::time::Duration;
use std::io::{self, BufRead};

fn main() {
    let (tx, rx) = mpsc::channel();
    let timeout_duration = Duration::from_secs(15); // 超时时长设置为15秒

    // 单独的线程来读取标准输入
    thread::spawn(move || {
        let stdin = io::stdin();
        let reader = stdin.lock();
        for line in reader.lines() {
            let line = line.expect("无法读取行");
            if tx.send(line).is_err() {
                break; // 如果接收方已经关闭，结束发送
            }
        }
    });

    // 主线程作为接收方
    loop {
        match rx.recv_timeout(timeout_duration) {
            Ok(line) => {
                println!("接收到数据: {}", line);
                // 这里处理接收到的数据
            },
            Err(mpsc::RecvTimeoutError::Timeout) => {
                println!("超过15秒未接收到数据，可能是阶段性结束。");
                // 这里可以处理阶段性结束的逻辑，如暂停操作或特定的处理
                // 如果确定整个数据流结束，可以退出循环
                // break;
            },
            Err(e) => {
                println!("接收错误: {:?}", e);
                break; // 其他错误，退出循环
            }
        }
    }
}
