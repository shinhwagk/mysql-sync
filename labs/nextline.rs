use std::io::{self, BufRead};

fn process_lines(line: &str, next_line: Option<&str>, state: &mut State) {
    if let Some(next) = next_line {
        println!("Next line is: {} {}", line, next);
    } else {
        println!("Next line is: {} {}", line, "none");
    }

    // 在这里对state进行修改
    state.counter += 1;
}

struct State {
    counter: usize,
}

fn main() {
    let stdin = io::stdin();
    let stdin_lock = stdin.lock();
    let mut lines = stdin_lock.lines().peekable();

    let mut state = State { counter: 0 };

    while let Some(line_result) = lines.next() {
        let line = match line_result {
            Ok(ln) => ln,
            Err(e) => {
                eprintln!("Failed to read line: {}", e);
                continue;
            }
        };

        let next_line = lines.peek().map(|res| res.as_ref().expect("Failed to peek line").as_str());
        process_lines(&line, next_line, &mut state);
    }

    println!("Processed lines: {}", state.counter);
}
