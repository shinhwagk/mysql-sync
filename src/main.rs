use std::collections::HashMap;
use std::io::{self, BufRead};
use std::sync::mpsc;
use std::thread;
use std::time::{Duration, Instant};

struct BinlogStatistics {
    update_rows: HashMap<String, u32>,
    delete_rows: HashMap<String, u32>,
    write_rows: HashMap<String, u32>,
    transaction: u32,
}

#[derive(Clone)]
struct TableColumnMap {
    name: String,
    types: String,
    // not_null: bool,
    // primary_key: bool,
}

#[derive(Clone)]
struct TableMap {
    map_num: String,
    database_name: String,
    table_name: String,
    cols: Vec<TableColumnMap>,
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

#[derive(PartialEq)]
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
    QueryXid, // same for Query
    TableMap,
    RotateTo,
    Stop,
    PreviousGTIDs,
    RowsQuery,
}

#[derive(PartialEq)]
enum BinlogEventTableMap {
    None,
    Columns,
    Skip,
}

enum BinlogEventWriteRows {
    None,
    Binlog,
}

type BinlogEventUpdateRows = BinlogEventWriteRows;
type BinlogEventDeleteRows = BinlogEventWriteRows;

enum BinlogEventRows {
    None,
    Binlog,
    PseudoSql,
}
fn binary_to_set(binary_str: &str, set_options: &Vec<String>) -> String {
    let num = usize::from_str_radix(binary_str, 2).unwrap();

    set_options
        .iter()
        .enumerate()
        .filter(|&(i, _)| (num >> i) & 1 == 1)
        .map(|(_, name)| name.as_str())
        .collect::<Vec<&str>>()
        .join(",")
}

fn parse_col_types_set(input: &str) -> Vec<String> {
    let trimmed = input.trim_start_matches("SET(").trim_end_matches(')');

    let mut elements = Vec::new();
    let mut temp = String::new();
    let mut in_quotes = false;

    for c in trimmed.chars() {
        match c {
            '\'' if in_quotes => {
                in_quotes = false;
            }
            '\'' => {
                in_quotes = true;
            }
            ',' if !in_quotes => {
                if !temp.trim().is_empty() {
                    elements.push(temp.trim().to_string());
                    temp.clear();
                }
            }
            _ => {
                temp.push(c);
            }
        }
    }

    if !temp.trim().is_empty() {
        elements.push(temp.trim().to_string());
    }

    elements
}

fn parse_col_types_enum(input: &str) -> Vec<String> {
    let trimmed = input.trim_start_matches("ENUM(").trim_end_matches(')');

    let mut elements = Vec::new();
    let mut temp = String::new();
    let mut in_quotes = false;

    for c in trimmed.chars() {
        match c {
            '\'' if in_quotes => {
                // 如果已经在引号内，则结束引号
                in_quotes = false;
            }
            '\'' => {
                in_quotes = true;
            }
            ',' if !in_quotes => {
                if !temp.trim().is_empty() {
                    elements.push(temp.trim().to_string());
                    temp.clear();
                }
            }
            _ => {
                temp.push(c);
            }
        }
    }

    if !temp.trim().is_empty() {
        elements.push(temp.trim().to_string());
    }

    elements
}

// fn process_lines(stdin_lock: std::io::StdinLock) -> Result<(), String> {
//     // eprintln!("{}", json_string);
//     // eprintln!("sdfsdfsd");
//     Ok(())
// }

fn parse_event_rows_pseudosql(psedosql: &Vec<String>, tmp_cache_event_table_map: &TableMap) -> Result<Vec<String>, String> {
    let mut statement: Vec<String> = Vec::new();

    for sql_str in psedosql {
        if sql_str.starts_with("###   @") {
            let end = sql_str.find('=').unwrap();
            let cmt_start = sql_str.find("/*").unwrap();

            let ci: i32 = sql_str[7..end].parse().expect("Failed to parse the string into an integer");

            let val = sql_str[end + 1..cmt_start].trim();
            let col = &(tmp_cache_event_table_map.cols)[(ci - 1) as usize];
            // let cn = &cols[(ci - 1) as usize];

            if col.types.starts_with("VARCHAR")
                || col.types.starts_with("CHAR")
                || col.types == "INT"
                || col.types == "TINYINT"
                || col.types == "SMALLINT"
                || col.types == "BIGINT"
                || col.types == "MEDIUMINT"
                || col.types == "FLOAT"
                || col.types == "DOUBLE"
                || col.types.starts_with("DECIMAL")
                || col.types.starts_with("BIT")
                || col.types == "DATE"
                || col.types == "TIME"
                || col.types == "DATETIME"
                || col.types == "YEAR"
                || col.types == "BLOB"
                || col.types == "MEDIUMBLOB"
                || col.types == "TINYBLOB"
                || col.types == "LONGBLOB"
                || col.types == "TINYTEXT"
                || col.types == "MEDIUMTEXT"
                || col.types == "LONGTEXT"
                || col.types == "TEXT"
            {
                statement.push(format!("{}={},", col.name, val));
            } else if col.types.starts_with("ENUM(") {
                let format_enum = parse_col_types_enum(&col.types);

                match val.parse::<usize>() {
                    Ok(index) => {
                        // println!("format e {:?} {}", format_enum, index);
                        statement.push(format!("{}='{}',", col.name, format_enum[index - 1]));
                    }
                    Err(e) => return Err(e.to_string()),
                }
            } else if col.types.starts_with("SET(") {
                let format_set = parse_col_types_set(&col.types);

                let set_val = &val[2..val.len() - 1];

                let val = binary_to_set(set_val, &format_set);

                statement.push(format!("{}='{}',", col.name, val));
            } else if col.types == "TIMESTAMP" {
                statement.push(format!("{}=FROM_UNIXTIME({}),", col.name, val));
            } else {
                println!("error xxx {} {}", col.types, val);
                return Err("发生错误了！".into());

                // error
            }
        } else {
            statement.push((&sql_str[4..]).to_string());
        }
    }

    Ok((statement))
}

fn parse_event_table_map_column(col_str: &str) -> (String, String) {
    let start = col_str.find('`').unwrap();
    let end = col_str[start + 1..].find('`').unwrap();

    let col_name = &col_str[start..start + end + 2];
    let mut col_types = "";

    if let Some(start) = col_str.find("ENUM(") {
        let end = col_str[start..].find(')').unwrap() + start + 1;
        col_types = &col_str[start..end]
    } else if let Some(start) = col_str.find("SET(") {
        let end = col_str[start..].find(')').unwrap() + start + 1;
        col_types = &col_str[start..end]
    } else {
        let tokens: Vec<&str> = col_str.split_whitespace().collect();
        if tokens.len() >= 2 {
            col_types = tokens[1];
        } else {
            // error
        }
    }

    (col_name.to_string(), col_types.to_string())
}

struct ParseBinlogLines {
    binlog_state: BinlogState,
    binlog_event: BinlogEvent,
    binlog_event_rows: BinlogEventRows,
    binlog_event_table_map: BinlogEventTableMap,
    tmp_cache_binlog_event_table_map: TableMap,
    tmp_cache_binlog_event_rows_table_with_pseudo_sql: HashMap<String, Vec<Vec<String>>>,
    tmp_cache_binlog_event_query_pseudo_sql: Vec<String>,
    tmp_cache_binlog_event_query: Vec<String>,
    tmp_cache_binlog_event_query_xid_pseudo_sql: Vec<String>,
    tmp_cache_binlog_event_rows_pseudo_sql: Vec<String>,
    tmp_cache_binlog_event_table_map1: HashMap<String, TableMap>,

    args_apply: String,

    output: bool,
}

impl ParseBinlogLines {
    fn default() -> Self {
        ParseBinlogLines {
            binlog_state: BinlogState::Head,
            binlog_event: BinlogEvent::None,
            binlog_event_rows: BinlogEventRows::None,
            binlog_event_table_map: BinlogEventTableMap::None,
            tmp_cache_binlog_event_table_map: TableMap {
                map_num: String::new(),
                database_name: String::new(),
                table_name: String::new(),
                cols: Vec::new(),
            },
            tmp_cache_binlog_event_rows_table_with_pseudo_sql: HashMap::new(),
            tmp_cache_binlog_event_query: Vec::new(),
            tmp_cache_binlog_event_query_xid_pseudo_sql: Vec::new(),
            tmp_cache_binlog_event_query_pseudo_sql: Vec::new(),
            tmp_cache_binlog_event_rows_pseudo_sql: Vec::new(),
            args_apply: "pseudo_sql".to_string(), // raw pseudo_sql replace_pseudo_sql
            tmp_cache_binlog_event_table_map1: HashMap::new(),

            output: false,
        }
    }

    fn process_line(&mut self, line: &str, next_line: Option<&str>) -> Result<(), String> {
        if !line.starts_with("#") {
            // stdout
        }

        // stderr statistics
        // if last_time.elapsed() >= Duration::from_secs(1) {
        //     let total: u32 = binlog_statistics.write_rows.values().sum();

        //     let json_string = format!(
        //         r#"{{"Write_rows": {}, "Binlog_pos": {}, "Binlog_file": {}, "Commit":{}}}"#,
        //         total, binlog_pos, binlog_file, binlog_statistics.transaction
        //     );

        //     eprintln!("{}", json_string);
        //     eprintln!("insert_rows: {:?}", binlog_statistics.write_rows);
        //     eprintln!("update_rows: {:?}", binlog_statistics.update_rows);
        //     eprintln!("delete_rows: {:?}", binlog_statistics.delete_rows);
        //     last_time = Instant::now();
        // }

        // statistics
        if line.starts_with("# at ") {
            match self.binlog_event {
                BinlogEvent::TableMap => {}
                BinlogEvent::WriteRows | BinlogEvent::UpdateRows | BinlogEvent::DeleteRows => {}
                _ => {}
            }
            self.binlog_event = BinlogEvent::None;

            // tmp_cache_binlog_pos_start = line[5..].parse::<u128>().expect("");
        } else if line.starts_with("#2") {
            self.binlog_state = BinlogState::BinlogEvent;
            let binlog_event_tokens: Vec<&str> = line.split_whitespace().collect();

            // tmp_cache_binlog_pos_end = binlog_event_tokens[6].parse::<u128>().expect("");

            if binlog_event_tokens.len() > 9 && binlog_event_tokens[7] == "CRC32" {
                // let binlog_event = words[9];
                match binlog_event_tokens[9] {
                    "Start:" => {
                        // binlog_last_event = binlog_event;
                        self.binlog_event = BinlogEvent::Start
                    }
                    "Previous-GTIDs" => self.binlog_event = BinlogEvent::PreviousGTIDs,
                    "Stop" => self.binlog_event = BinlogEvent::Stop,
                    "GTID" => {
                        // #240429  2:41:42 server id 1  end_log_pos 234 CRC32 0xbb588611 	GTID	last_committed=0	sequence_number=1	rbr_only=no	original_committed_timestamp=1714358502099265	immediate_commit_timestamp=1714358502099265	transaction_length=181
                        self.binlog_event = BinlogEvent::Gtid;
                        // let last_commited = binlog_event_tokens[10]
                        //     .split("=")
                        //     .last()
                        //     .expect("No '=' found in token")
                        //     .parse::<u128>()
                        //     .expect("Failed to parse number");
                        // delay_commit = last_commited == tmp_cache_binlog_event_gtid_last_commited;

                        // eprintln!("last_committed={} sequence_number={}", binlog_event_tokens[10], binlog_event_tokens[11]);
                    }
                    "Xid" => {
                        // #240429  2:41:44 server id 1  end_log_pos 3040324 CRC32 0x4dd3b280 	Xid = 8
                        self.binlog_event = BinlogEvent::Xid;
                        // binlog_statistics.transaction += 1;
                    }
                    "Query" => {
                        // #240413  5:14:13 server id 161183306  end_log_pos 83777 CRC32 0x07d16955 	Query	thread_id=5200748	exec_time=0	error_code=0
                        // #240512  9:09:25 server id 1  end_log_pos 4620 CRC32 0xe0905a82 	Query	thread_id=49	exec_time=0	error_code=0	Xid = 148

                        if binlog_event_tokens.len() >= 14 && binlog_event_tokens[13] == "Xid" {
                            self.binlog_event = BinlogEvent::QueryXid;
                        } else {
                            self.binlog_event = BinlogEvent::Query;
                        }
                    }
                    "Write_rows:" => {
                        // #240429  2:41:42 server id 1  end_log_pos 1681 CRC32 0x3d5e28c2 	Write_rows: table id 104 flags: STMT_END_F
                        self.binlog_event = BinlogEvent::WriteRows;
                        // binlog_event_write_rows = BinlogEventWriteRows::None;
                        self.binlog_event_rows = BinlogEventRows::None;

                        if binlog_event_tokens.len() == 15 {
                            let table_id = binlog_event_tokens[12].parse::<u16>().expect("Write_rows parse error.");

                            // if table_id == binlog_event_tokens[12].to_string() {
                            //     *binlog_statistics.write_rows.entry(table_map.clone()).or_insert(0) += 1;
                            // } else {
                            //     // eprintln!("error: {}", line)
                            //     // error
                            // }
                        } else {
                            // error
                        }
                    }
                    "Delete_rows:" => {
                        // #240413 14:10:09 server id 161183306  end_log_pos 108664847 CRC32 0x219eafb4 	Delete_rows: table id 955
                        self.binlog_event = BinlogEvent::DeleteRows;
                        // binlog_event_delete_rows = BinlogEventDeleteRows::None;
                        self.binlog_event_rows = BinlogEventRows::None;

                        //     if binlog_event_tokens.len() == 15 {
                        //         if binlog_event_tokens[12].to_string() == table_id {
                        //             // *binlog_statistics.delete_rows.entry(table_map.clone()).or_insert(0) += 1;
                        //         } else {
                        //             // error
                        //         }
                        //     } else {
                        //         // error
                        //     }
                    }
                    "Update_rows:" => {
                        // #240413  5:14:13 server id 161183306  end_log_pos 2162 CRC32 0x6b65e044 	Update_rows: table id 394 flags: STMT_END_F
                        self.binlog_event = BinlogEvent::UpdateRows;
                        // binlog_event_update_rows = BinlogEventUpdateRows::None;
                        self.binlog_event_rows = BinlogEventRows::None;

                        // if binlog_event_tokens.len() == 15 {
                        //     if binlog_event_tokens[12].to_string() == table_id {
                        //         *binlog_statistics.update_rows.entry(table_map.clone()).or_insert(0) += 1;
                        //     } else {
                        //         // error
                        //     }
                        // } else {
                        //     // error
                        // }
                    }
                    "Table_map:" => {
                        // #240413  5:14:13 server id 161183306  end_log_pos 1628 CRC32 0x33192119 	Table_map: `merchant_center_vela_v1`.`v3_express_recommend` mapped to number 394
                        self.binlog_event = BinlogEvent::TableMap;
                        self.binlog_event_table_map = BinlogEventTableMap::None;

                        let num = binlog_event_tokens[14];

                        let db_tab: Vec<&str> = binlog_event_tokens[10].split('.').collect();
                        if db_tab.len() == 2 {
                            let database_name = db_tab[0].to_string();
                            let table_name = db_tab[1].to_string();

                            // if database_name == "`test`" {
                            //     self.binlog_event_table_map = BinlogEventTableMap::Skip;
                            //     return;
                            // }

                            self.tmp_cache_binlog_event_table_map.map_num = num.to_string();
                            self.tmp_cache_binlog_event_table_map.database_name = database_name;
                            self.tmp_cache_binlog_event_table_map.table_name = table_name;
                            self.tmp_cache_binlog_event_table_map.cols.clear();
                        } else {
                            // error
                        }
                    }
                    "Rotate" if binlog_event_tokens.get(10) == Some(&"to") => {
                        // #240427  5:14:18 server id 161933306  end_log_pos 1073747122 CRC32 0xd6395e4f 	Rotate to mysql-bin.000031  pos: 4
                        self.binlog_event = BinlogEvent::RotateTo;
                    }
                    "Rows_query" => {
                        self.binlog_event = BinlogEvent::RowsQuery;
                    }
                    _ => {
                        // error
                    }
                }
            } else {
                // error
            }
            // skip event head line
            return Ok(());
        } else if line.starts_with("#691231") || line.starts_with("#700101") {
            let tokens: Vec<&str> = line.split_whitespace().collect();
            match tokens.len() {
                12 => {} //binlog_file = tokens[9].to_string(),
                14 => {} //binlog_file = tokens[11].to_string(),
                _ => {
                    // error
                }
            }
            self.binlog_event = BinlogEvent::SysRotateTo
        } else {
        }

        match self.binlog_state {
            BinlogState::BinlogEvent => match self.binlog_event {
                BinlogEvent::Query => {
                    if line.starts_with("SET ") {
                        self.tmp_cache_binlog_event_query.push(line.to_string());
                        if line.starts_with("SET TIMESTAMP=") && line.ends_with("/*!*/;") {
                            // let timestamp_str = line.strip_prefix("SET TIMESTAMP=").and_then(|s| s.strip_suffix("'/*!*/;"));
                            // match timestamp_str {
                            //     Some(timestamp) => match timestamp.parse::<u32>() {
                            //         Ok(ts) => binlog_timestamp = ts,
                            //         Err(_) => eprintln!("'{}' is not a valid u32", timestamp),
                            //     },
                            //     None => {
                            //         // error
                            //     }
                            // }
                            self.tmp_cache_binlog_event_query.push(line.to_string());
                        }
                    } else if line == "BEGIN" {
                        self.tmp_cache_binlog_event_query.push(line.to_string());
                        self.tmp_cache_binlog_event_query_pseudo_sql.push("BEGIN;".to_string());
                    } else if line == "/*!*/;" {
                        self.tmp_cache_binlog_event_query.push(line.to_string());
                        // self.tmp_cache_binlog_event_query_raw.push(line);
                        if let Some(last) = self.tmp_cache_binlog_event_query_xid_pseudo_sql.last_mut() {
                            last.push_str(";");
                        }
                    } else if line.starts_with("/*!") {
                        self.tmp_cache_binlog_event_query.push(line.to_string());
                    } else if line.starts_with("# original_commit_timestamp=") {
                    } else if line.starts_with("# immediate_commit_timestamp=") {
                    } else if line.starts_with("use `") {
                        // binlog entry: use `database_1`/*!*/;
                        self.tmp_cache_binlog_event_query_xid_pseudo_sql.push(line.replace("/*!*/", ""));
                        // self.tmp_cache_binlog_event_query_raw.push(line.to_string());
                    } else {
                        self.tmp_cache_binlog_event_query_xid_pseudo_sql.push(line.to_string());

                        // error
                    }
                }
                BinlogEvent::QueryXid => {
                    if line.starts_with("SET ") && line.ends_with("/*!*/;") {
                    } else if line == "/*!*/;" {
                        if let Some(last) = self.tmp_cache_binlog_event_query_xid_pseudo_sql.last_mut() {
                            last.push_str(";");
                        }
                        self.output = true;
                    } else if line.starts_with("/*!") {
                    } else if line.starts_with("use `") {
                        // binlog entry: use `database_1`/*!*/;
                        self.tmp_cache_binlog_event_query_xid_pseudo_sql.push(line.replace("/*!*/", ""));
                    } else {
                        self.tmp_cache_binlog_event_query_xid_pseudo_sql.push(line.to_string());
                    }
                }
                BinlogEvent::TableMap => {
                    // # has_generated_invisible_primary_key=0
                    // # Columns(`a` INT UNSIGNED NOT NULL,
                    // #         `b` ENUM NOT NULL)
                    match self.binlog_event_table_map {
                        BinlogEventTableMap::Columns => {
                            // append column
                            let col_str = &line[10..line.len() - 1];

                            let (col_name, col_types) = parse_event_table_map_column(col_str);
                            let col = TableColumnMap {
                                name: col_name,
                                types: col_types,
                            };

                            self.tmp_cache_binlog_event_table_map.cols.push(col);

                            if line.ends_with(")") {
                                self.tmp_cache_binlog_event_table_map1.insert(
                                    self.tmp_cache_binlog_event_table_map.map_num.clone(),
                                    self.tmp_cache_binlog_event_table_map.clone(),
                                );
                                self.binlog_event_table_map = BinlogEventTableMap::None
                            }
                        }
                        BinlogEventTableMap::None => {
                            if line.starts_with("# Columns(") {
                                self.binlog_event_table_map = BinlogEventTableMap::Columns;
                                let col_str = &line[10..line.len() - 1];

                                let (col_name, col_types) = parse_event_table_map_column(col_str);

                                let col = TableColumnMap {
                                    name: col_name,
                                    types: col_types,
                                };

                                self.tmp_cache_binlog_event_table_map.cols.push(col);
                                if line.ends_with(")") {
                                    self.tmp_cache_binlog_event_table_map1.insert(
                                        self.tmp_cache_binlog_event_table_map.map_num.clone(),
                                        self.tmp_cache_binlog_event_table_map.clone(),
                                    );

                                    self.binlog_event_table_map = BinlogEventTableMap::None
                                    // tmp_cache_event_rows_mult_pseudo_sql.push(tmp_c)
                                }
                            } else if line.starts_with("# Primary Key(") {
                            } else {
                                // error
                            }
                        }
                        BinlogEventTableMap::Skip => {
                            return Ok(());
                        }
                    }
                }
                // BinlogEvent::WriteRows => {
                //     /*
                //     BINLOG '
                //     5ggvZhMBAAAAOwAAAGgGAAAAAGgAAAAAAAEABW15c3FsAAl0aW1lX3pvbmUAAgP+AvcBAAEBgHkO
                //     8IQ=
                //     5ggvZh4BAAAAKQAAAJEGAAAAAGgAAAAAAAEAAgAC/wACAAAAAsIoXj0=
                //     '/*!*/;
                //     ### INSERT INTO `mysql`.`time_zone`
                //     ### SET
                //     ###   @1=2 /* INT meta=0 nullable=0 is_null=0 */
                //     ###   @2=2 /* ENUM(1 byte) meta=63233 nullable=0 is_null=0 */
                //     */
                //     match binlog_event_write_rows {
                //         BinlogEventWriteRows::Binlog => {
                //             if line == "'/*!*/;" {
                //                 binlog_event_write_rows = BinlogEventWriteRows::None
                //             }
                //         }
                //         BinlogEventWriteRows::None => {
                //             if line.starts_with("### INSERT INTO ") {
                //                 binlog_event_write_rows = BinlogEventWriteRows::None;

                //                 temp_cache_rows_event_pseudo_sql.push(line);

                //                 // append column
                //             } else if line == "BINLOG '" {
                //                 binlog_event_write_rows = BinlogEventWriteRows::Binlog
                //             } else {
                //             }
                //         }
                //     }
                // }
                BinlogEvent::WriteRows | BinlogEvent::DeleteRows | BinlogEvent::UpdateRows => {
                    if self.binlog_event_table_map == BinlogEventTableMap::Skip {
                        return Ok(());
                    }

                    match self.binlog_event_rows {
                        BinlogEventRows::Binlog => {
                            if line == "'/*!*/;" {
                                self.binlog_event_rows = BinlogEventRows::None;

                                // self.tmp_cache_binlog_event_rows_binlog.push(line);

                                let db = self.tmp_cache_binlog_event_table_map.database_name.clone();
                                let tab = self.tmp_cache_binlog_event_table_map.table_name.clone();

                                // self.tmp_cache_binlog_event_rows_table_with_binlog
                                //     .insert(format!("`{}`.`{}`", db, tab), self.tmp_cache_binlog_event_rows_binlog.clone());

                                // self.tmp_cache_binlog_event_rows_binlog.clear();
                            } else {
                                // self.tmp_cache_binlog_event_rows_binlog.push(line);
                            }
                        }
                        BinlogEventRows::PseudoSql => {
                            if line.starts_with("### DELETE FROM ") || line.starts_with("### UPDATE ") || line.starts_with("### INSERT INTO ") {
                                let map_num = self.tmp_cache_binlog_event_table_map.map_num.clone();
                                self.tmp_cache_binlog_event_rows_table_with_pseudo_sql
                                    .entry(map_num.clone())
                                    .or_insert(Vec::new())
                                    .push(self.tmp_cache_binlog_event_rows_pseudo_sql.clone());
                                self.tmp_cache_binlog_event_rows_pseudo_sql.clear();

                                self.tmp_cache_binlog_event_rows_pseudo_sql.push(line.to_string());
                            } else if line.starts_with("### ") {
                                self.tmp_cache_binlog_event_rows_pseudo_sql.push(line.to_string());
                                if let Some(next) = next_line {
                                    if next.starts_with("# ") {
                                        let map_num = self.tmp_cache_binlog_event_table_map.map_num.clone();
                                        self.tmp_cache_binlog_event_rows_table_with_pseudo_sql
                                            .entry(map_num)
                                            .or_insert(Vec::new())
                                            .push(self.tmp_cache_binlog_event_rows_pseudo_sql.clone());
                                        self.tmp_cache_binlog_event_rows_pseudo_sql.clear();
                                    }
                                }
                            } else {
                                // error
                            }
                        }
                        BinlogEventRows::None => {
                            if line.starts_with("### DELETE FROM ") || line.starts_with("### UPDATE ") || line.starts_with("### INSERT INTO ") {
                                self.binlog_event_rows = BinlogEventRows::PseudoSql;

                                self.tmp_cache_binlog_event_rows_pseudo_sql.push(line.to_string());
                            } else if line == "BINLOG '" {
                                self.binlog_event_rows = BinlogEventRows::Binlog;
                                // self.tmp_cache_binlog_event_rows_binlog.push(line);
                            } else if line == "" {
                                // empty line
                            } else {
                                // error
                            }
                        }
                    }
                }
                // BinlogEvent::UpdateRows => {
                //     match binlog_event_update_rows {
                //         BinlogEventUpdateRows::Binlog => {
                //             if line == "'/*!*/;" {
                //                 binlog_event_update_rows = BinlogEventUpdateRows::None
                //             }
                //         }
                //         BinlogEventUpdateRows::None => {
                //             if line.starts_with("### UPDATE ") {
                //                 binlog_event_update_rows = BinlogEventUpdateRows::None;
                //                 temp_cache_rows_event_pseudo_sql.push(line);
                //                 // append column
                //             } else if line == "BINLOG '" {
                //                 binlog_event_update_rows = BinlogEventUpdateRows::Binlog
                //             } else {
                //             }
                //         }
                //     }
                // }
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
                        match line.strip_prefix("SET @@SESSION.GTID_NEXT= '").and_then(|s| s.strip_suffix("'/*!*/;")) {
                            Some(gtid_next) => {
                                // gtid = gtid_next.to_string();
                                // cache_stdout.push(format!("# gtid: {}", gtid));
                            }
                            None => {
                                // error
                            }
                        }
                        // self.tmp_cache_binlog_event_gtid.push(line);
                    } else if line.starts_with("# original_commit_timestamp=") {
                    } else if line.starts_with("# immediate_commit_timestamp=") {
                    } else if line.starts_with("/*!800") {
                        // self.tmp_cache_binlog_event_gtid.push(line);
                    } else if line.starts_with("/*!507") {
                        // self.tmp_cache_binlog_event_gtid.push(line);
                    } else {
                        // error
                    }
                }
                BinlogEvent::Start => {
                    // #240429  2:41:49 server id 1  end_log_pos 126 CRC32 0x2db1350e 	Start: binlog v 4, server v 8.0.36 created 240429  2:41:49 at startup
                }
                BinlogEvent::RotateTo => {}
                BinlogEvent::Xid => {
                    // binlog entry: COMMIT/*!*/;
                    if line == "COMMIT/*!*/;" {
                        // self.tmp_cache_binlog_event_xid.push(line);
                        self.output = true;
                    } else {
                        // error
                    }
                }
                BinlogEvent::PreviousGTIDs => {
                    // #240429  2:41:49 server id 1  end_log_pos 197 CRC32 0x01882c96 	Previous-GTIDs
                    // # [empty]
                    if line == "# [empty]" {
                    } else {
                        // error
                    }
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
                BinlogEvent::RowsQuery => {
                    // match line.strip_prefix("# ") {
                    //     Some(token) => sql_statement2.push(token.to_string()),
                    //     None => {
                    //         // error
                    //     }
                    // }
                }
                BinlogEvent::None => {}
                _ => {
                    // error
                }
            },
            BinlogState::Head => {
                if line == "# The proper term is pseudo_replica_mode, but we use this compatibility alias" {
                } else if line == "" {
                } else if line.starts_with("/*!") {
                    // self.tmp_cache_binlog_head.push(line);
                } else if line.starts_with("DELIMITER ") {
                    // self.binlog_delimiter = (&line[10..line.len() - 1]).to_string();
                    self.binlog_state = BinlogState::BinlogEvent;
                    // self.tmp_cache_binlog_head.push(line);
                } else {
                    // error
                }
            }
        }

        if self.output {
            if self.args_apply == "raw" {
                // for output in &self.tmp_cache_binlog_head {
                //     println!("{}", output);
                // }

                // for output in &self.tmp_cache_binlog_event_gtid {
                //     println!("{}", output);
                // }
                // for output in &self.tmp_cache_binlog_event_query {
                //     println!("{}", output);
                // }

                // // for sql in &tmp_cache_binlog_event_data_rows {
                // //     println!("xxxx {}", sql)
                // // }
                // for (dbtab, binlogs) in &self.tmp_cache_binlog_event_rows_table_with_binlog {
                //     // for binlog in binlogs {
                //     if !dbtab.starts_with("`test`") {
                //         for binlog in binlogs {
                //             println!("{}", binlog)
                //         }
                //     }
                // }
                // for output in &self.tmp_cache_binlog_event_xid {
                //     println!("{}", output);
                // }
            } else if self.args_apply == "replace_pseudo_sql" {
            } else if self.args_apply == "pseudo_sql" {
                if self.tmp_cache_binlog_event_query_xid_pseudo_sql.len() >= 1 {
                    for pseudo_sql in &self.tmp_cache_binlog_event_query_xid_pseudo_sql {
                        println!("{}", pseudo_sql);
                    }
                    self.tmp_cache_binlog_event_query_xid_pseudo_sql.clear();
                }

                if self.tmp_cache_binlog_event_rows_table_with_pseudo_sql.len() >= 1 {
                    println!("BEGIN;");
                    for (map_num, pseudo_sqls) in &self.tmp_cache_binlog_event_rows_table_with_pseudo_sql {
                        let table_map = self.tmp_cache_binlog_event_table_map1.get(map_num).unwrap();

                        // if table_map.database_name != "`test`" {
                        for pseudo_sql in pseudo_sqls {
                            match parse_event_rows_pseudosql(pseudo_sql, table_map) {
                                Ok((pseudosql)) => {
                                    println!("{}", format!("{};", pseudosql.join(" ").trim_end_matches(',')))
                                }
                                Err(e) => return Err(format!("column type parse faile: {}", e)),
                            }
                        }
                        // }
                    }
                    println!("COMMIT;");
                    self.tmp_cache_binlog_event_rows_table_with_pseudo_sql.clear();
                }
            }

            // self.tmp_cache_binlog_head.clear();
            // self.tmp_cache_binlog_event_gtid.clear();
            // self.tmp_cache_binlog_event_query.clear();
            // self.tmp_cache_binlog_event_xid.clear();
            // self.tmp_cache_binlog_event_rows_table_with_binlog.clear();
            // self.tmp_cache_binlog_event_table_map

            self.output = false;
        }
        Ok(())
    }
}

fn process_lines() {
    let (tx, rx) = mpsc::channel();
    let timeout_duration: Duration = Duration::from_millis(100);

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
            if tx.send(line).is_err() {
                println!("Sending error, receiver has likely quit.");
                break;
            }
        }
    });

    let mut current_line: Option<String> = None;
    let mut processor = ParseBinlogLines::default();

    loop {
        match rx.recv_timeout(timeout_duration) {
            Ok(line) => {
                if let Some(current) = current_line.take() {
                    match processor.process_line(&current, Some(&line)) {
                        Ok(()) => {}
                        Err(e) => {
                            eprintln!("操作失败：{}", e);
                            std::process::exit(1);
                        }
                    }
                }
                current_line = Some(line);
            }
            Err(mpsc::RecvTimeoutError::Timeout) => {
                if let Some(current) = current_line.take() {
                    match processor.process_line(&current, None) {
                        Ok(()) => {}
                        Err(e) => {
                            eprintln!("操作失败：{}", e);
                            std::process::exit(1);
                        }
                    }
                }

                thread::sleep(Duration::from_millis(10));
                continue;
            }
            Err(e) => {
                eprintln!("Receive error: {:?}", e);
                break;
            }
        }
    }
}

fn main() {
    process_lines()
}
