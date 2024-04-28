use std::collections::HashMap;
use std::io::{self, BufRead};

enum BinlogState {
    Normal,
    Binlog,
    SqlText,
    // Pos,
}
struct Event_base {}

struct Event_gtid {
    ev_base: Event_base,
    ev_data: String,
    last_committed: u32,
    sequence_number: u32,
    rbr_only: String,
    original_committed_timestamp: u32,
    immediate_commit_timestamp: u32,
    transaction_length: u32,
}

struct Event_gtid_data {
    server_uuid: String,
    tid: u32,
}

struct Event_query {
    ev_base: Event_base,
    ev_data: String,
    thread_id: u16,
    exec_time: u16,
    error_code: u8,
    xid: Option<u16>,
}
use std::time::{Duration, Instant};

fn process_line(stdin_lock: std::io::StdinLock) {
    let mut update_rows: HashMap<String, i32> = HashMap::new();
    let mut delete_rows: HashMap<String, i32> = HashMap::new();
    let mut write_rows: HashMap<String, i32> = HashMap::new();

    let mut binlog_state = BinlogState::Normal;

    let mut table_map: String = String::new();
    let mut table_id: String = String::new();

    let mut binlog_pos: String = String::new();
    let mut binlog_file: String = String::new();

    let mut commit: u32 = 0;

    let mut last_time = Instant::now();

    for line_result in stdin_lock.lines() {
        match line_result {
            Ok(line) => {
                if !line.starts_with("#") {
                    println!("{}", line);
                }

                // if last_time.elapsed() >= Duration::from_secs(1) {
                let total: i32 = write_rows.values().sum();

                let json_string = format!(
                    r#"{{"Write_rows": {}, "Binlog_pos": {}, "Binlog_file": {}, "Commit":{}}}"#,
                    total, binlog_pos, binlog_file, commit
                );

                eprintln!("{}", json_string);
                //     last_time = Instant::now();
                // }
                loop {
                    match binlog_state {
                        BinlogState::Normal => {
                            if line.starts_with("# at ") {
                                let pos = &line[5..];
                                binlog_pos = pos.to_string();
                                // eprintln!("{}", binlog_pos)
                            } else if line.starts_with("#2") {
                                let words: Vec<&str> = line.split_whitespace().collect();

                                if words.len() >= 10 && words[7] == "CRC32" {
                                    match words[9] {
                                        "Start:" | "Previous-GTIDs" | "Stop" | "GTID" => {}
                                        "Xid" => {}
                                        "Query" => {
                                            if words.len() == 13 {
                                                // eprintln!("111 {:?}", words)
                                                commit += 1
                                            } else if words.len() == 16 {
                                                // eprintln!("xid {}", words[13])
                                            }
                                        }
                                        "Write_rows:" => {
                                            if words[12].to_string() == table_id {
                                                *write_rows
                                                    .entry(table_map.clone())
                                                    .or_insert(0) += 1;
                                            }
                                        }
                                        "Delete_rows:" => {
                                            *delete_rows.entry(table_map.clone()).or_insert(0) += 1;
                                        }
                                        "Update_rows:" => {
                                            *update_rows.entry(table_map.clone()).or_insert(0) += 1;
                                        }
                                        "Table_map:" => {
                                            table_map = words[10].replace("`", "");
                                            table_id = words[14].to_string();
                                        }
                                        "Rotate" if words.get(10) == Some(&"to") => {
                                            // eprintln!("222 {}", words[11])
                                        }
                                        _ => {}
                                    }
                                } else {
                                    eprintln!("{}", line)
                                }
                            } else if line.starts_with("#7") {
                                let words: Vec<&str> = line.split_whitespace().collect();

                                if words[7] == "CRC32"
                                    && words[9] == "Rotate"
                                    && words.get(10) == Some(&"to")
                                {
                                    binlog_file = words[11].to_string()
                                    // eprintln!("{}", line)
                                } else if words[7] == "Rotate" && words.get(8) == Some(&"to") {
                                    // eprintln!("222 {}", words[9])
                                    binlog_file = words[9].to_string()
                                } else {
                                }
                            } else if line == "BINLOG '" {
                                binlog_state = BinlogState::Binlog;
                            } else if line.starts_with("###") {
                                binlog_state = BinlogState::SqlText;
                                continue;
                            } else if line.starts_with("SET @@SESSION.GTID_NEXT= '")
                                && line.ends_with("'/*!*/;")
                            {
                            } else {
                            }
                            break;
                        }
                        BinlogState::Binlog => {
                            if line == "'/*!*/;" {
                                binlog_state = BinlogState::Normal;
                            } else {
                            }
                            break;
                        }
                        BinlogState::SqlText => {
                            if line.starts_with("###") {
                            } else {
                                binlog_state = BinlogState::Normal;
                                continue;
                            }
                            break;
                        }
                    }
                }
            }
            Err(e) => {
                eprintln!("Error reading line: {}", e);
                break;
            }
        }
    }
    eprintln!("write_rows: {:?}", write_rows);
    eprintln!("update_rows: {:?}", update_rows);
    eprintln!("delete_rows: {:?}", delete_rows);

    let total: i32 = write_rows.values().sum();

    let json_string = format!(
        r#"{{"Write_rows": {}, "Binlog_pos": {}, "Binlog_file": {}, "Commit":{}}}"#,
        total, binlog_pos, binlog_file, commit
    );
    eprintln!("{}", json_string);
}

fn main() {
    let stdin = io::stdin();
    let stdin_lock = stdin.lock();

    process_line(stdin_lock)
}
