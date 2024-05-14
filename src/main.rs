use std::collections::HashMap;
use std::io::{self, BufRead};

use std::time::{Duration, Instant};

struct BinlogStatistics {
    update_rows: HashMap<String, u32>,
    delete_rows: HashMap<String, u32>,
    write_rows: HashMap<String, u32>,
    transaction: u32,
}

struct BinlogDml {
    gtid: String,
    last_committed: u32,
    sequence_number: u32,
    log_file: String,
    log_pos: u32,
    row_events: Vec<i32>,
}

struct UpdateRowsEvent {}
struct WriteRowsEvent {}
struct DeleteRowsEvent {}
struct TableMapEvent {}

struct BinlogTableMap {
    mapped_number: u8,
}

enum BinlogRowsKind {
    write_rows,
    update_rows,
    delete_rows,
}

struct BinlogEventDataDeleteRows {
    tab_id: u8,
    binlog: Vec<Vec<String>>,
}

struct BinlogEventDataRows {
    tab_id: u8,
    binlog: Vec<Vec<String>>,
    pseudo_sql: Vec<String>,
}

struct BinlogRows {
    tab_map: BinlogTableMap,
    binlog: Vec<Vec<String>>,
    db_name: String,
    tab_name: String,
    rows: Vec<Vec<String>>,
    raws: Vec<String>,
}

struct TableColumnMap {
    name: String,
    types: String,
    // not_null: bool,
    // primary_key: bool,
}

struct TableMap {
    map_num: u8,
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

enum BinlogEventTableMap {
    None,
    Columns,
}

enum BinlogEventWriteRows {
    None,
    Binlog,
}

type BinlogEventUpdateRows = BinlogEventWriteRows;
type BinlogEventDeleteRows = BinlogEventWriteRows;

#[derive(Debug)]
enum BinlogEventRows {
    None,
    Binlog,
    PseudoSql,
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

// process mysql data type 'set'
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

fn parse_event_rows_pseudosql(psedosql: &Vec<String>, tmp_cache_event_table_map: &TableMap) -> Vec<String> {
    let mut statement: Vec<String> = Vec::new();

    for sql_str in psedosql {
        if sql_str.starts_with("###   @") {
            let end = sql_str.find('=').unwrap();
            let cmt_start = sql_str.find("/*").unwrap();

            let ci: i32 = sql_str[7..end]
                .parse()
                .expect("Failed to parse the string into an integer");

            let val = sql_str[end + 1..cmt_start].trim();
            let col = &(tmp_cache_event_table_map.cols)[(ci - 1) as usize];
            // let cn = &cols[(ci - 1) as usize];

            if col.types.starts_with("VARCHAR")
                || col.types.starts_with("CHAR")
                || col.types == "INT"
                || col.types == "SMALLINT"
                || col.types == "BIGINT"
                || col.types == "FLOAT"
                || col.types == "DOUBLE"
                || col.types.starts_with("DECIMAL")
                || col.types == "DATE"
                || col.types == "TIME"
                || col.types == "DATETIME"
                || col.types == "YEAR"
                || col.types == "BLOB"
                || col.types == "TEXT"
            {
                statement.push(format!("{}={},", col.name, val));
            } else if col.types.starts_with("ENUM(") {
                let format_enum = parse_col_types_enum(&col.types);

                match val.parse::<usize>() {
                    Ok(index) => {
                        statement.push(format!("{}='{}',", col.name, format_enum[index]));
                    }
                    Err(_) => {
                        println!("Failed to parse index");
                    }
                }
            } else if col.types.starts_with("SET(") {
                let format_set = parse_col_types_set(&col.types);

                let set_val = &val[2..val.len() - 1];

                let val = binary_to_set(set_val, &format_set);

                statement.push(format!("{}='{}',", col.name, val));
            } else if col.types == "TIMESTAMP" {
                statement.push(format!("{}=FROM_UNIXTIME({}),", col.name, val));
            } else {
                // error
            }
        } else {
            statement.push((&sql_str[4..]).to_string());
        }
    }
    statement
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

fn process_lines(stdin_lock: std::io::StdinLock) -> Result<(), String> {
    let mut version_identifier: HashMap<String, String> = HashMap::new();

    let mut binlog_statistics = BinlogStatistics {
        delete_rows: HashMap::new(),
        write_rows: HashMap::new(),
        update_rows: HashMap::new(),
        transaction: 0,
    };

    version_identifier.insert("8.0.34-26".to_string(), "#691231".to_string());
    version_identifier.insert("8.0.36".to_string(), "#700101".to_string());

    let args_raw = false;

    let mut binlog_state = BinlogState::Head;
    let mut binlog_event = BinlogEvent::None;
    let mut binlog_last_event = BinlogEvent::None;

    let mut table_map: String = String::new();
    let mut table_id: String = String::new();

    let mut binlog_pos: String = String::new();
    let mut binlog_file: String = String::new();

    let mut last_time = Instant::now();

    let mut binlog_delimiter: String = String::new();
    let mut gtid: String = String::new();

    let mut sql_statement2: Vec<String> = Vec::new();

    let mut binlog_timestamp: u32 = 0;
    let mut binlog_event_timestamp: u16 = 0;

    let mut binlog_event_table_map: BinlogEventTableMap = BinlogEventTableMap::None;
    let mut binlog_event_write_rows: BinlogEventWriteRows = BinlogEventWriteRows::None;
    let mut binlog_event_update_rows: BinlogEventUpdateRows = BinlogEventUpdateRows::None;
    let mut binlog_event_delete_rows: BinlogEventDeleteRows = BinlogEventDeleteRows::None;

    let mut binlog_event_rows: BinlogEventRows = BinlogEventRows::None;

    // let mut column_names = Vec::new();

    let mut cache_columns_str: Vec<String> = Vec::new();

    let mut cache_table_definition: HashMap<u8, TableMap> = HashMap::new();

    let mut output = false;

    let mut filter_tables: Vec<(String, String)> = Vec::new();

    let mut cache_stdout: Vec<String> = Vec::new();

    let mut tmp_cache_event_rows_binlog: Vec<String> = Vec::new();
    let mut tmp_cache_event_rows_pseudo_sql: Vec<String> = Vec::new();
    let mut tmp_cache_event_table_map: TableMap = TableMap {
        map_num: 0,
        database_name: String::new(),
        table_name: String::new(),
        cols: Vec::new(),
    };

    let mut tmp_cache_event_rows_mult_pseudo_sql: Vec<Vec<String>> = Vec::new();
    let mut tmp_cache_event_rows_mult_binlog: Vec<Vec<String>> = Vec::new();

    let mut tmp_cache_binlog_event_data_rows = BinlogEventDataRows {
        tab_id: 0,
        binlog: Vec::new(),
        pseudo_sql: Vec::new(),
    };

    // let mut temp_cache_event_query

    for line_result in stdin_lock.lines() {
        let mut binlog_event_tokens: Vec<&str> = Vec::new();

        match line_result {
            Ok(line) => {
                if !line.starts_with("#") {
                    // stdout
                    if args_raw {
                        println!("{}", line);
                    }
                }

                // stderr statistics
                if last_time.elapsed() >= Duration::from_secs(1) {
                    let total: u32 = binlog_statistics.write_rows.values().sum();

                    let json_string = format!(
                        r#"{{"Write_rows": {}, "Binlog_pos": {}, "Binlog_file": {}, "Commit":{}}}"#,
                        total, binlog_pos, binlog_file, binlog_statistics.transaction
                    );

                    eprintln!("{}", json_string);
                    eprintln!("insert_rows: {:?}", binlog_statistics.write_rows);
                    eprintln!("update_rows: {:?}", binlog_statistics.update_rows);
                    eprintln!("delete_rows: {:?}", binlog_statistics.delete_rows);
                    last_time = Instant::now();
                }

                // statistics
                if line.starts_with("# at ") {
                    match binlog_event {
                        BinlogEvent::TableMap => {}
                        BinlogEvent::WriteRows | BinlogEvent::UpdateRows | BinlogEvent::DeleteRows => {
                            tmp_cache_event_rows_mult_pseudo_sql.push(tmp_cache_event_rows_pseudo_sql.clone());
                            tmp_cache_event_rows_pseudo_sql.clear();
                            // if tmp_cache_binlog_event_data_rows.tab_id == tmp_cache_event_table_map.map_num && tmp_cache_event_table_map.database_name
                            for ps in &tmp_cache_event_rows_mult_pseudo_sql {
                                let statement: Vec<String> = parse_event_rows_pseudosql(ps, &tmp_cache_event_table_map);
                                // cache_stdout.push(;
                                tmp_cache_binlog_event_data_rows
                                    .pseudo_sql
                                    .push(format!("{};", statement.join(" ").trim_end_matches(',')));
                            }
                            tmp_cache_event_rows_mult_pseudo_sql.clear();

                            for b in &tmp_cache_event_rows_mult_binlog {
                                tmp_cache_binlog_event_data_rows.binlog.push(b.clone());
                            }
                            tmp_cache_event_rows_mult_binlog.clear();
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
                            "Start:" => {
                                // binlog_last_event = binlog_event;
                                binlog_event = BinlogEvent::Start
                            }
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
                                // #240512  9:09:25 server id 1  end_log_pos 4620 CRC32 0xe0905a82 	Query	thread_id=49	exec_time=0	error_code=0	Xid = 148

                                if binlog_event_tokens.len() >= 14 && binlog_event_tokens[13] == "Xid" {
                                    binlog_event = BinlogEvent::QueryXid;
                                } else {
                                    binlog_event = BinlogEvent::Query;
                                    tmp_cache_binlog_event_data_rows = BinlogEventDataRows {
                                        tab_id: 0,
                                        binlog: Vec::new(),
                                        pseudo_sql: Vec::new(),
                                    };
                                }
                            }
                            "Write_rows:" => {
                                // #240429  2:41:42 server id 1  end_log_pos 1681 CRC32 0x3d5e28c2 	Write_rows: table id 104 flags: STMT_END_F
                                binlog_event = BinlogEvent::WriteRows;
                                binlog_event_write_rows = BinlogEventWriteRows::None;
                                binlog_event_rows = BinlogEventRows::None;

                                tmp_cache_event_rows_binlog.clear();

                                if binlog_event_tokens.len() == 15 {
                                    tmp_cache_binlog_event_data_rows.tab_id =
                                        binlog_event_tokens[12].parse::<u8>().expect("xxx");

                                    if binlog_event_tokens[12].to_string() == table_id {
                                        *binlog_statistics.write_rows.entry(table_map.clone()).or_insert(0) += 1;
                                    } else {
                                        // eprintln!("error: {}", line)
                                        // error
                                    }
                                } else {
                                    // error
                                }
                            }
                            "Delete_rows:" => {
                                // #240413 14:10:09 server id 161183306  end_log_pos 108664847 CRC32 0x219eafb4 	Delete_rows: table id 955
                                binlog_event = BinlogEvent::DeleteRows;
                                binlog_event_delete_rows = BinlogEventDeleteRows::None;
                                binlog_event_rows = BinlogEventRows::None;

                                // tmp_cache_event_rows_pseudo_sql.clear();
                                tmp_cache_event_rows_binlog.clear();

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
                                binlog_event_update_rows = BinlogEventUpdateRows::None;
                                binlog_event_rows = BinlogEventRows::None;

                                // tmp_cache_event_rows_pseudo_sql.clear();
                                tmp_cache_event_rows_binlog.clear();

                                if binlog_event_tokens.len() == 15 {
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
                                binlog_event_table_map = BinlogEventTableMap::None;

                                match binlog_event_tokens[14].parse() {
                                    Ok(num) => {
                                        if num != tmp_cache_event_table_map.map_num {
                                            tmp_cache_event_table_map = TableMap {
                                                map_num: 0,
                                                database_name: String::new(),
                                                table_name: String::new(),
                                                cols: Vec::new(),
                                            };
                                            tmp_cache_event_table_map.map_num = num
                                        } else {
                                            binlog_event = BinlogEvent::None;
                                        }
                                    }
                                    Err(e) => println!("Error converting to an integer: {}", e),
                                }

                                let db_tab: Vec<&str> = binlog_event_tokens[10].split('.').collect();
                                if db_tab.len() == 2 {
                                    tmp_cache_event_table_map.database_name = db_tab[0].to_string();
                                    tmp_cache_event_table_map.table_name = db_tab[1].to_string();
                                } else {
                                    // error
                                }
                            }
                            "Rotate" if binlog_event_tokens.get(10) == Some(&"to") => {
                                // #240427  5:14:18 server id 161933306  end_log_pos 1073747122 CRC32 0xd6395e4f 	Rotate to mysql-bin.000031  pos: 4
                                binlog_event = BinlogEvent::RotateTo;
                            }
                            "Rows_query" => {
                                binlog_event = BinlogEvent::RowsQuery;
                            }
                            _ => {
                                // error
                            }
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
                        BinlogEvent::Query => {
                            if line.starts_with("SET TIMESTAMP=") && line.ends_with("/*!*/;") {
                                let timestamp_str = line
                                    .strip_prefix("SET TIMESTAMP=")
                                    .and_then(|s| s.strip_suffix("'/*!*/;"));
                                match timestamp_str {
                                    Some(timestamp) => match timestamp.parse::<u32>() {
                                        Ok(ts) => binlog_timestamp = ts,
                                        Err(_) => eprintln!("'{}' is not a valid u32", timestamp),
                                    },
                                    None => {
                                        // error
                                    }
                                }
                            } else if line == "BEGIN" {
                                cache_stdout.push("BEGIN;".to_string());
                            } else {
                            }
                        }
                        BinlogEvent::QueryXid => {
                            if line.starts_with("SET ") && line.ends_with("/*!*/;") {
                            } else if line.starts_with("/*!") {
                            } else if line == "/*!*/;" {
                                output = true;
                            } else if line.starts_with("use `") {
                                // binlog entry: use `database_1`/*!*/;

                                cache_stdout.push(line.replace("/*!*/", ""));
                            } else {
                                // tmp_cache_event_rows_pseudo_sql.push(line.trim().to_string());
                            }
                        }
                        BinlogEvent::TableMap => {
                            // # has_generated_invisible_primary_key=0
                            // # Columns(`a` INT UNSIGNED NOT NULL,
                            // #         `b` ENUM NOT NULL)
                            match binlog_event_table_map {
                                BinlogEventTableMap::Columns => {
                                    // append column
                                    let col_str = &line[10..line.len() - 1];

                                    let (col_name, col_types) = parse_event_table_map_column(col_str);
                                    let col = TableColumnMap {
                                        name: col_name,
                                        types: col_types,
                                    };

                                    tmp_cache_event_table_map.cols.push(col);

                                    if line.ends_with(")") {
                                        binlog_event_table_map = BinlogEventTableMap::None
                                    }
                                }
                                BinlogEventTableMap::None => {
                                    if line.starts_with("# Columns(") {
                                        binlog_event_table_map = BinlogEventTableMap::Columns;
                                        let col_str = &line[10..line.len() - 1];

                                        let (col_name, col_types) = parse_event_table_map_column(col_str);
                                        let col = TableColumnMap {
                                            name: col_name,
                                            types: col_types,
                                        };

                                        tmp_cache_event_table_map.cols.push(col);
                                        if line.ends_with(")") {
                                            binlog_event_table_map = BinlogEventTableMap::None
                                            // tmp_cache_event_rows_mult_pseudo_sql.push(tmp_c)
                                        }
                                    }
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
                            match binlog_event_rows {
                                BinlogEventRows::Binlog => {
                                    if line == "'/*!*/;" {
                                        binlog_event_rows = BinlogEventRows::None;
                                        tmp_cache_event_rows_binlog.push(line);
                                        tmp_cache_event_rows_mult_binlog.push(tmp_cache_event_rows_binlog.clone());
                                        tmp_cache_event_rows_binlog.clear();
                                    } else {
                                        tmp_cache_event_rows_binlog.push(line);
                                    }
                                }
                                BinlogEventRows::PseudoSql => {
                                    if line.starts_with("### DELETE FROM ")
                                        || line.starts_with("### UPDATE ")
                                        || line.starts_with("### INSERT INTO ")
                                    {
                                        tmp_cache_event_rows_mult_pseudo_sql
                                            .push(tmp_cache_event_rows_pseudo_sql.clone());
                                        tmp_cache_event_rows_pseudo_sql.clear();

                                        tmp_cache_event_rows_pseudo_sql.push(line);
                                    } else if line.starts_with("### ") {
                                        tmp_cache_event_rows_pseudo_sql.push(line);
                                    } else {
                                        // error
                                    }
                                }
                                BinlogEventRows::None => {
                                    if line.starts_with("### DELETE FROM ")
                                        || line.starts_with("### UPDATE ")
                                        || line.starts_with("### INSERT INTO ")
                                    {
                                        tmp_cache_event_rows_pseudo_sql.clear();

                                        binlog_event_rows = BinlogEventRows::PseudoSql;
                                        tmp_cache_event_rows_pseudo_sql.push(line.clone());
                                    } else if line == "BINLOG '" {
                                        binlog_event_rows = BinlogEventRows::Binlog;
                                        tmp_cache_event_rows_binlog.clear();
                                        tmp_cache_event_rows_binlog.push(line);
                                    } else if line == "" {
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
                                match line
                                    .strip_prefix("SET @@SESSION.GTID_NEXT= '")
                                    .and_then(|s| s.strip_suffix("'/*!*/;"))
                                {
                                    Some(gtid_next) => {
                                        gtid = gtid_next.to_string();
                                        cache_stdout.push(format!("# gtid: {}", gtid));
                                    }
                                    None => {
                                        // error
                                    }
                                }
                            } else if line.starts_with("# original_commit_timestamp=") {
                            } else if line.starts_with("# immediate_commit_timestamp=") {
                            } else if line.starts_with("/*!80001 SET @@session.") {
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
                                cache_stdout.push("COMMIT;".replace("/*!*/", "").to_string());
                                output = true;
                            } else {
                                // error
                            }
                            // table_id = "".to_string();
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
                            match line.strip_prefix("# ") {
                                Some(token) => sql_statement2.push(token.to_string()),
                                None => {
                                    // error
                                }
                            }
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
                        } else if line.starts_with("DELIMITER ") {
                            binlog_delimiter = (&line[10..line.len() - 1]).to_string();
                            binlog_state = BinlogState::BinlogEvent
                        } else {
                            // error
                        }
                    }
                }

                if output {
                    if tmp_cache_binlog_event_data_rows.tab_id == tmp_cache_event_table_map.map_num
                        && tmp_cache_event_table_map.database_name != "`test`"
                    {
                        for sql in &tmp_cache_binlog_event_data_rows.pseudo_sql {
                            println!("xxxx {}", sql)
                        }
                    }

                    // for binlogs in &tmp_cache_binlog_event_data_rows.binlog {
                    //     for binlog in binlogs {
                    //         println!("xxxx {}", binlog)
                    //     }
                    // }

                    output = false;
                }

                // output
                // if !args_raw {
                //     for output in cache_stdout.clone() {
                //         println!("{}", output);
                //     }
                //     cache_stdout.clear();
                // }
            }
            Err(e) => {
                eprintln!("Error reading line: {}", e);
                break;
            }
        }
    }
    // eprintln!("insert_rows: {:?}", binlog_statistics.write_rows);
    // eprintln!("update_rows: {:?}", binlog_statistics.update_rows);
    // eprintln!("delete_rows: {:?}", binlog_statistics.delete_rows);

    let total: u32 = binlog_statistics.write_rows.values().sum();

    let json_string = format!(
        r#"{{"Write_rows": {}, "Binlog_pos": {}, "Binlog_file": {}, "Commit":{}}}"#,
        total, binlog_pos, binlog_file, binlog_statistics.transaction
    );
    // eprintln!("{}", json_string);
    // eprintln!("sdfsdfsd");
    Ok(())
}

fn main() {
    let stdin = io::stdin();
    let stdin_lock = stdin.lock();

    match process_lines(stdin_lock) {
        Ok(_) => {}
        Err(e) => eprintln!("错误: {}", e),
    }
}
