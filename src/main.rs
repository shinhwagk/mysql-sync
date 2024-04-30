use std::collections::HashMap;
use std::io::{self, BufRead};
use std::time::{Duration, Instant};

struct BinlogStatistics {
    update_rows: HashMap<String, u32>,
    delete_rows: HashMap<String, u32>,
    write_rows: HashMap<String, u32>,
    transaction: u32,
}

// fn print_person(person: &Person) {
//     println!("Name: {}, Age: {}", person.name, person.age);
// }

// fn main() {
//     let person = Person {
//         name: "Alice".to_string(),
//         age: 30,
//     };

//     print_person(&person);
// }

enum BinlogState {
    Head,
    // None,
    // Binlog,
    BinlogEvent,
    // SqlText,
    // Gtid, // Pos,
}

enum BinlogEvent {
    SysRotateTo,
    None,
    Gtid,
    Start,
    Xid,
    DeleteRows,
    UpdateRows,
    WriteRows,
    Query,
    TableMap,
    RotateTo,
    Stop,
    PreviousGTIDs,
}

// struct Event_base {}

// struct Event_gtid {
//     ev_base: Event_base,
//     ev_data: String,
//     last_committed: u32,
//     sequence_number: u32,
//     rbr_only: String,
//     original_committed_timestamp: u32,
//     immediate_commit_timestamp: u32,
//     transaction_length: u32,
// }

// struct Event_gtid_data {
//     server_uuid: String,
//     tid: u32,
// }

// struct Event_query {
//     ev_base: Event_base,
//     ev_data: String,
//     thread_id: u16,
//     exec_time: u16,
//     error_code: u8,
//     xid: Option<u16>,
// }

fn process_line(stdin_lock: std::io::StdinLock) -> Result<(), String> {
    let mut version_identifier: HashMap<String, String> = HashMap::new();

    let mut binlog_statistics = BinlogStatistics {
        delete_rows: HashMap::new(),
        write_rows: HashMap::new(),
        update_rows: HashMap::new(),
        transaction: 0,
    };

    version_identifier.insert("8.0.34-26".to_string(), "#691231".to_string());
    version_identifier.insert("8.0.36".to_string(), "#700101".to_string());

    let mut binlog_state = BinlogState::Head;
    let mut binlog_event = BinlogEvent::None;

    let mut table_map: String = String::new();
    let mut table_id: String = String::new();

    let mut binlog_pos: String = String::new();
    let mut binlog_file: String = String::new();

    let mut last_time = Instant::now();

    let mut binlog_delimiter: String = String::new();
    let mut gtid: String = String::new();

    let mut sql_statement: Vec<String> = Vec::new();

    for line_result in stdin_lock.lines() {
        let mut binlog_event_tokens: Vec<&str> = Vec::new();

        match line_result {
            Ok(line) => {
                if !line.starts_with("#") {
                    // stdout
                    println!("{}", line);
                }

                // stderr statistics
                if last_time.elapsed() >= Duration::from_secs(1) {
                    let total: u32 = binlog_statistics.write_rows.values().sum();

                    let json_string = format!(
                        r#"{{"Write_rows": {}, "Binlog_pos": {}, "Binlog_file": {}, "Commit":{}}}"#,
                        total, binlog_pos, binlog_file, binlog_statistics.transaction
                    );

                    eprintln!("{}", json_string);
                    last_time = Instant::now();
                }

                // statistics
                if line.starts_with("# at ") {
                    match binlog_event {
                        BinlogEvent::UpdateRows | BinlogEvent::DeleteRows | BinlogEvent::WriteRows => {
                            // eprintln!("{:?}", sql_statement)
                        }
                        _ => {}
                    }
                    binlog_event = BinlogEvent::None;

                    let pos = &line[5..];
                    binlog_pos = pos.to_string();
                } else if line.starts_with("#2") {
                    binlog_state = BinlogState::BinlogEvent;
                    binlog_event_tokens = line.split_whitespace().collect();

                    if binlog_event_tokens.len() > 9 && binlog_event_tokens[7] == "CRC32" {
                        // let binlog_event = words[9];
                        match binlog_event_tokens[9] {
                            "Start:" => binlog_event = BinlogEvent::Start,
                            "Previous-GTIDs" => binlog_event = BinlogEvent::PreviousGTIDs,
                            "Stop" => binlog_event = BinlogEvent::Stop,
                            "GTID" => {
                                // #240429  2:41:42 server id 1  end_log_pos 234 CRC32 0xbb588611 	GTID	last_committed=0	sequence_number=1	rbr_only=no	original_committed_timestamp=1714358502099265	immediate_commit_timestamp=1714358502099265	transaction_length=181
                                binlog_event = BinlogEvent::Gtid;
                            }
                            "Xid" => {
                                // #240429  2:41:44 server id 1  end_log_pos 3040324 CRC32 0x4dd3b280 	Xid = 8
                                binlog_event = BinlogEvent::Xid;

                                binlog_statistics.transaction += 1;
                            }
                            "Query" => {
                                // #240413  5:14:13 server id 161183306  end_log_pos 83777 CRC32 0x07d16955 	Query	thread_id=5200748	exec_time=0	error_code=0
                                binlog_event = BinlogEvent::Query;
                            }
                            "Write_rows:" => {
                                // #240429  2:41:42 server id 1  end_log_pos 1681 CRC32 0x3d5e28c2 	Write_rows: table id 104 flags: STMT_END_F
                                binlog_event = BinlogEvent::WriteRows;

                                if binlog_event_tokens.len() == 15 {
                                    if binlog_event_tokens[12].to_string() == table_id {
                                        *binlog_statistics.write_rows.entry(table_map.clone()).or_insert(0) += 1;
                                    } else {
                                        eprintln!("error: {}", line)
                                        // error
                                    }
                                } else {
                                    // error
                                }
                            }
                            "Delete_rows:" => {
                                // #240413 14:10:09 server id 161183306  end_log_pos 108664847 CRC32 0x219eafb4 	Delete_rows: table id 955
                                binlog_event = BinlogEvent::DeleteRows;

                                if binlog_event_tokens.len() == 15 {
                                    if binlog_event_tokens[12].to_string() == table_id {
                                        *binlog_statistics.delete_rows.entry(table_map.clone()).or_insert(0) += 1;
                                    } else {
                                        // error
                                    }
                                } else {
                                    // error
                                }
                            }
                            "Update_rows:" => {
                                // #240413  5:14:13 server id 161183306  end_log_pos 2162 CRC32 0x6b65e044 	Update_rows: table id 394 flags: STMT_END_F
                                binlog_event = BinlogEvent::UpdateRows;

                                if binlog_event_tokens.len() == 1 {
                                    if binlog_event_tokens[12].to_string() == table_id {
                                        *binlog_statistics.update_rows.entry(table_map.clone()).or_insert(0) += 1;
                                    } else {
                                        // error
                                    }
                                } else {
                                    // error
                                }
                            }
                            "Table_map:" => {
                                // #240413  5:14:13 server id 161183306  end_log_pos 1628 CRC32 0x33192119 	Table_map: `merchant_center_vela_v1`.`v3_express_recommend` mapped to number 394
                                binlog_event = BinlogEvent::TableMap;

                                table_map = binlog_event_tokens[10].replace("`", "");
                                table_id = binlog_event_tokens[14].to_string();
                            }
                            "Rotate" if binlog_event_tokens.get(10) == Some(&"to") => {
                                // #240427  5:14:18 server id 161933306  end_log_pos 1073747122 CRC32 0xd6395e4f 	Rotate to mysql-bin.000031  pos: 4
                                binlog_event = BinlogEvent::RotateTo;
                            }
                            _ => {}
                        }
                    } else {
                        // error
                    }
                    continue;
                } else if line.starts_with("#691231") || line.starts_with("#700101") {
                    let tokens: Vec<&str> = line.split_whitespace().collect();
                    match tokens.len() {
                        12 => binlog_file = tokens[9].to_string(),
                        14 => binlog_file = tokens[11].to_string(),
                        _ => {
                            // error
                        }
                    }
                    binlog_event = BinlogEvent::SysRotateTo
                } else {
                }

                match binlog_state {
                    BinlogState::BinlogEvent => match binlog_event {
                        BinlogEvent::Query => {}
                        BinlogEvent::TableMap => {
                            // # has_generated_invisible_primary_key=0
                            // # Columns(INT UNSIGNED NOT NULL,
                            // #         ENUM NOT NULL)
                        }
                        BinlogEvent::WriteRows => {
                            /*
                            BINLOG '
                            5ggvZhMBAAAAOwAAAGgGAAAAAGgAAAAAAAEABW15c3FsAAl0aW1lX3pvbmUAAgP+AvcBAAEBgHkO
                            8IQ=
                            5ggvZh4BAAAAKQAAAJEGAAAAAGgAAAAAAAEAAgAC/wACAAAAAsIoXj0=
                            '/*!*/;
                            ### INSERT INTO `mysql`.`time_zone`
                            ### SET
                            ###   @1=2 /* INT meta=0 nullable=0 is_null=0 */
                            ###   @2=2 /* ENUM(1 byte) meta=63233 nullable=0 is_null=0 */
                            */

                            if line.starts_with("### ") {
                                match line.strip_prefix("### ") {
                                    Some(token) => sql_statement.push(token.to_string()),
                                    None => {
                                        // error
                                    }
                                }
                            }
                        }
                        BinlogEvent::DeleteRows => {
                            if line.starts_with("### ") {
                                match line.strip_prefix("### ") {
                                    Some(token) => sql_statement.push(token.to_string()),
                                    None => {
                                        // error
                                    }
                                }
                            }
                        }
                        BinlogEvent::UpdateRows => {
                            if line.starts_with("### ") {
                                match line.strip_prefix("### ") {
                                    Some(token) => sql_statement.push(token.to_string()),
                                    None => {
                                        // error
                                    }
                                }
                            }
                        }
                        BinlogEvent::Gtid => {
                            /*
                                # original_commit_timestamp=1714358502099265 (2024-04-29 02:41:42.099265 UTC)
                                # immediate_commit_timestamp=1714358502099265 (2024-04-29 02:41:42.099265 UTC)
                                /*!80001 SET @@session.original_commit_timestamp=1714358502099265*//*!*/;
                                /*!80014 SET @@session.original_server_version=80036*//*!*/;
                                /*!80014 SET @@session.immediate_server_version=80036*//*!*/;
                                SET @@SESSION.GTID_NEXT= 'fae48e3d-05d1-11ef-9575-0242ac140002:1'/*!*/;
                            */

                            if line.starts_with("SET @@SESSION.GTID_NEXT= '") && line.ends_with("'/*!*/;") {
                                match line
                                    .strip_prefix("SET @@SESSION.GTID_NEXT= '")
                                    .and_then(|s| s.strip_suffix("'/*!*/;"))
                                {
                                    Some(gtid_next) => gtid = gtid_next.to_string(),
                                    None => {
                                        // error
                                    }
                                }
                            }
                        }
                        BinlogEvent::Start => {
                            // #240429  2:41:49 server id 1  end_log_pos 126 CRC32 0x2db1350e 	Start: binlog v 4, server v 8.0.36 created 240429  2:41:49 at startup
                        }
                        BinlogEvent::RotateTo => {}
                        BinlogEvent::Xid => {

                            // table_id = "".to_string();
                        }
                        BinlogEvent::PreviousGTIDs => {
                            // #240429  2:41:49 server id 1  end_log_pos 197 CRC32 0x01882c96 	Previous-GTIDs
                        }
                        BinlogEvent::Stop => {
                            // #240429  2:41:45 server id 1  end_log_pos 3040347 CRC32 0x6360bdb4 	Stop
                        }

                        BinlogEvent::SysRotateTo => {
                            // #691231 16:00:00 server id 1  end_log_pos 0 	Rotate to mysql-bin.000029  pos: 4
                            // #691231 16:00:00 server id 1  end_log_pos 0 CRC32 0x6bdc0ec3 	Rotate to mysql-bin.000030  pos: 4
                            // #700101  0:00:00 server id 1  end_log_pos 0 	Rotate to mysql-bin.000001  pos: 4
                            // #700101  0:00:00 server id 1  end_log_pos 0 CRC32 0xf1a55fa5 	Rotate to mysql-bin.000002  pos: 4
                            // binlog entry: SET @@SESSION.GTID_NEXT= 'AUTOMATIC' /* added by mysqlbinlog */ /*!*/;
                        }
                        BinlogEvent::None => {}
                    },
                    BinlogState::Head => {
                        if line == "# The proper term is pseudo_replica_mode, but we use this compatibility alias" {
                        } else if line == "" {
                        } else if line.starts_with("/*!") {
                        } else if line.starts_with("DELIMITER ") {
                            binlog_delimiter = (&line[10..line.len() - 1]).to_string();
                            binlog_state = BinlogState::BinlogEvent
                        } else {
                            // error
                        }
                    }
                }

                //                         // if line.starts_with("# at ") {
                //                         //     let pos = &line[5..];
                //                         //     binlog_pos = pos.to_string();
                //                         //     binlog_event = BinlogEvent::None
                //                         //     // eprintln!("{}", binlog_pos)
                //                         // } else if   line.starts_with("#2") {
                //                         //     let words: Vec<&str> = line.split_whitespace().collect();

                //                         //     if words.len() >= 10 && words[7] == "CRC32" {
                //                         //         let binlog_event = words[9];
                //                         //         match binlog_event {
                //                         //             "Start:" | "Previous-GTIDs" | "Stop" => {}
                //                         //             "GTID" => {
                //                         //                 binlog_state = BinlogState::Gtid;
                //                         //             }
                //                         //             "Xid" => {
                //                         //                 stats_commit += 1;
                //                         //                 table_id = "".to_string();
                //                         //             }
                //                         //             "Query" => {
                //                         //                 if words.len() == 13 {
                //                         //                     // eprintln!("111 {:?}", words)
                //                         //                     // commit += 1
                //                         //                 } else if words.len() == 16 {
                //                         //                     // eprintln!("xid {}", words[13])
                //                         //                 }
                //                         //             }
                //                         //             "Write_rows:" => {
                //                         //                 if words[12].to_string() == table_id {
                //                         //                     *write_rows
                //                         //                         .entry(table_map.clone())
                //                         //                         .or_insert(0) += 1;
                //                         //                 }
                //                         //             }
                //                         //             "Delete_rows:" => {
                //                         //                 *delete_rows.entry(table_map.clone()).or_insert(0) += 1;
                //                         //             }
                //                         //             "Update_rows:" => {
                //                         //                 *update_rows.entry(table_map.clone()).or_insert(0) += 1;
                //                         //             }
                //                         //             "Table_map:" => {
                //                         //                 table_map = words[10].replace("`", "");
                //                         //                 table_id = words[14].to_string();
                //                         //             }
                //                         //             "Rotate" if words.get(10) == Some(&"to") => {
                //                         //                 // eprintln!("222 {}", words[11])
                //                         //             }
                //                         //             _ => {}
                //                         //         }
                //                         //     } else {
                //                         //         eprintln!("{}", line)
                //                         //     }
                //                         // } else if line.starts_with("#700101") {
                //                         //     let words: Vec<&str> = line.split_whitespace().collect();

                //                         //     if words[7] == "CRC32"
                //                         //         && words[9] == "Rotate"
                //                         //         && words.get(10) == Some(&"to")
                //                         //     {
                //                         //         // binlog entry: #700101  0:00:00 server id 1  end_log_pos 0 CRC32 0xf1a55fa5 	Rotate to mysql-bin.000002  pos: 4

                //                         //         binlog_file = words[11].to_string()
                //                         //         // eprintln!("{}", line)
                //                         //     } else if words[7] == "Rotate" && words.get(8) == Some(&"to") {
                //                         //         // binlog entry: #700101  0:00:00 server id 1  end_log_pos 0 	Rotate to mysql-bin.000001  pos: 4

                //                         //         // eprintln!("222 {}", words[9])
                //                         //         binlog_file = words[9].to_string()
                //                         //     } else {
                //                         //     }
                //                         // } else if line == "BINLOG '" {
                //                         //     binlog_state = BinlogState::Binlog;
                //                         // } else if line.starts_with("###") {
                //                         //     binlog_state = BinlogState::SqlText;
                //                         //     continue;
                //                         // } else {
                //                         // }
                //                         // break;
                //                     }
                //                     // BinlogState::Binlog => {
                //                     //     if line == "'/*!*/;" {

                //                     //     } else {
                //                     //     }
                //                     //     break;
                //                     // }
                //                     // BinlogState::SqlText => {
                //                     //     if line.starts_with("###") {
                //                     //     } else {

                //                     //         continue;
                //                     //     }
                //                     //     break;
                //                     // }
                //                     // BinlogState::Gtid => {
                //                     //     if line.starts_with("SET @@SESSION.GTID_NEXT= '")
                //                     //         && line.ends_with("'/*!*/;")
                //                     //     {
                //                     //         // eprintln!("{}", line);
                //                     //         // binlog_state = BinlogState::None;
                //                     //     }
                //                     //     break;
                //                     // }
                //                     BinlogState::Head => {
                //                         if line == "# The proper term is pseudo_replica_mode, but we use this compatibility alias" {

                //                         }  else if line.starts_with("/*!") {}

                //                         }else if  line.starts_with("DELIMITER"){
                //                             delimiter =( &line[10..line.len() - 1]).to_string();
                //                             binlog_state=BinlogState::BinlogEvent
                //                         }
                //                     }
                //                 }
            }
            Err(e) => {
                eprintln!("Error reading line: {}", e);
                break;
            }
        }
    }
    eprintln!("write_rows: {:?}", binlog_statistics.write_rows);
    eprintln!("update_rows: {:?}", binlog_statistics.update_rows);
    eprintln!("delete_rows: {:?}", binlog_statistics.delete_rows);

    let total: u32 = binlog_statistics.write_rows.values().sum();

    let json_string = format!(
        r#"{{"Write_rows": {}, "Binlog_pos": {}, "Binlog_file": {}, "Commit":{}}}"#,
        total, binlog_pos, binlog_file, binlog_statistics.transaction
    );
    eprintln!("{}", json_string);
    Ok(())
}

fn main() {
    let stdin = io::stdin();
    let stdin_lock = stdin.lock();

    match process_line(stdin_lock) {
        Ok(_) => println!("处理成功完成。"),
        Err(e) => eprintln!("错误: {}", e),
    }
}
