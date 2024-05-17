use std::io::{self, BufRead};
use std::time::{Duration, Instant};

fn main() {
    let stdin = io::stdin();
    let mut lines = stdin.lock().lines();

    let mut last_line = None;
    let timeout_duration = Duration::new(5, 0); // 设置超时时间为5秒
    let mut last_time = Instant::now();

    while let Some(line) = lines.next() {
        let line = line.expect("Failed to read line");
        let now = Instant::now();

        if last_time.elapsed() > timeout_duration {
            println!("curr: {:?} next: None", last_line);
            last_line = None;
        }

        if let Some(l) = &last_line {
            println!("curr: {:?} next: {:?}", l, line);
        }

        last_line = Some(line);
        last_time = now;
    }

    if last_line.is_some() {
        println!("curr: {:?} next: None", last_line);
    }
    println!("xxxx")
}
