use std::sync::mpsc;
use std::thread;
use std::time::Duration;
use std::io::{self, BufRead};

fn main() {
    let (tx, rx) = mpsc::channel();
    let timeout_duration = Duration::from_millis(100); // 超时时长设置为100毫秒

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

    let mut current_line = None;
    // 主线程作为接收方
    loop {
        match rx.recv_timeout(timeout_duration) {
            Ok(line) => {
                if let Some(current) = current_line.take() {
                    process_line(&current, Some(&line));
                }
                current_line = Some(line);
            },
            Err(mpsc::RecvTimeoutError::Timeout) => {
                if let Some(current) = current_line.take() {
                    process_line(&current, None);
                }
                break; // 超时则假定结束，退出循环
            },
            Err(e) => {
                println!("接收错误: {:?}", e);
                break; // 其他错误，退出循环
            }
        }
    }
}

// Function to process the current line and optionally the next line
fn process_line(current_line: &str, next_line: Option<&str>) {
    match next_line {
        Some(next) => println!("Current Line: {}, Next Line: {}", current_line, next),
        None => println!("Current Line: {}, No more data received", current_line),
    }
}
