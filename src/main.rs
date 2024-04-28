use std::collections::HashMap;
use std::io::{self, BufRead};

enum BinlogState {
    Normal,
    Binlog,
    SqlText,
}

fn process_line(stdin_lock: std::io::StdinLock) {
    let mut update_rows: HashMap<String, i32> = HashMap::new();
    let mut delete_rows: HashMap<String, i32> = HashMap::new();
    let mut write_rows: HashMap<String, i32> = HashMap::new();

    let mut binlog_state = BinlogState::Normal;

    let mut table_map: String = String::new();
    let mut table_id: String = String::new();

    for line_result in stdin_lock.lines() {
        match line_result {
            Ok(line) => {
                if !line.starts_with("#") {
                    println!("{}", line);
                }

                match binlog_state {
                    BinlogState::Normal => {
                        if line.starts_with("# at ") {
                            let pos = &line[5..];
                            // eprintln!("{}", pos)
                        } else if line.starts_with("#2") {
                            let words: Vec<&str> = line.split_whitespace().collect();

                            if words[7] == "CRC32" {
                                match words[9] {
                                    "Start:" | "Previous-GTIDs" | "Stop" | "GTID"   =>{}
                                    "Xid" => {

                                    }
                                    "Query" && words[13] =="Xid" =>{}
                                    "Write_rows:" => {
                                        if words[12].to_string() == table_id {
                                            *write_rows.entry(table_map.clone()).or_insert(0) += 1;
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
                                    "Rotate" => {
                                        eprintln!("{}", line)
                                    }
                                    _ => {}
                                }
                            } else {
                                eprintln!("{}", line)
                            }
                        } else if line.starts_with("#7") {
                            let words: Vec<&str> = line.split_whitespace().collect();

                            if words[7] == "CRC32" && words[9] == "Rotate" {
                                eprintln!("{}", line)
                            } else if words[7] == "Rotate" {
                                eprintln!("{}", line)
                            } else {
                            }
                        } else if line == "BINLOG '" {
                            binlog_state = BinlogState::Binlog;
                        } else if line.starts_with("###") {
                            binlog_state = BinlogState::SqlText;
                            continue;
                        } else if line.starts_with("SET @@SESSION.GTID_NEXT= '")  and line.ends_with("'/*!*/;") {

                        } else {
                        }
                    }
                    BinlogState::Binlog => {
                        if line == "'/*!*/;" {
                            binlog_state = BinlogState::Normal;
                        } else {
                        }
                    }
                    BinlogState::SqlText => {
                        if line.starts_with("###") {
                        } else {
                            binlog_state = BinlogState::Normal;
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
}

fn main() {
    let stdin = io::stdin();
    let stdin_lock = stdin.lock();

    process_line(stdin_lock)
}
