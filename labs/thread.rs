use std::io::{self, BufRead};
use std::sync::mpsc;
use std::thread;
use std::time::Duration;

fn main() {
    let (tx, rx) = mpsc::channel();
    let timeout_duration = Duration::from_millis(1000);

    thread::spawn(move || {
        let stdin = io::stdin();
        let reader = stdin.lock();
        for line in reader.lines() {
            let line = match line {
                Ok(l) => l,
                Err(e) => {
                    eprintln!("Error reading line: {:?}", e);
                    break;
                }
            };
            println!("{}", line);
            if tx.send(line).is_err() {
                println!("Sending error, receiver has likely quit.");
                break;
            }
        }
    });

    let mut current_line: Option<String> = None;

    loop {
        match rx.recv_timeout(timeout_duration) {
            Ok(line) => {
                if let Some(current) = current_line.take() {
                    process_line(&current, Some(&line));
                }
                current_line = Some(line);
            }
            Err(mpsc::RecvTimeoutError::Timeout) => {
                if let Some(current) = current_line.take() {
                    process_line(&current, None);
                }

                thread::sleep(Duration::from_millis(10));
                continue;
            }
            Err(e) => {
                println!("Receive error: {:?}", e);
                break;
            }
        }
    }
}

fn process_line(current_line: &str, next_line: Option<&str>) {
    match next_line {
        Some(next) => println!("Current Line: ###{}###, Next Line: ###{}###", current_line, next),
        None => println!("Current Line: ###{}###, No more data received", current_line),
    }
}
