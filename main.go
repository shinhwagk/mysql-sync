package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-mysql-org/go-mysql/replication"
	"github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"

	_ "github.com/go-sql-driver/mysql"
)

func (a Canal) handleQueryEvent(e *replication.QueryEvent) error {
	parser := parser.New()
	stmts, warns, err := parser.Parse(string(e.Query), "", "")
	for _, warn := range warns {
		fmt.Println(warn)
	}
	if err != nil {
		return err
	}
	for _, stmt := range stmts {
		switch t := stmt.(type) {
		case *ast.RenameTableStmt:
		case *ast.AlterTableStmt:
			schema := string(e.Schema)
			if len(schema) == 0 {
				schema = t.Table.Schema.O
			}
			a.ch <- MysqlOperationDDLTable{schema: schema, table: t.Table.Name.O, query: string(e.Query)}
		case *ast.DropTableStmt:
			schema := string(e.Schema)
			for _, tab := range t.Tables {
				if len(schema) == 0 {
					schema = tab.Schema.O
				}
				a.ch <- MysqlOperationDDLTable{schema: schema, table: tab.Name.O, query: string(e.Query)}
			}
		case *ast.CreateTableStmt:
			schema := string(e.Schema)
			if len(schema) == 0 {
				schema = t.Table.Schema.O
			}
			a.ch <- MysqlOperationDDLTable{schema: schema, table: t.Table.Name.O, query: string(e.Query)}
		case *ast.TruncateTableStmt:
			schema := string(e.Schema)
			if len(schema) == 0 {
				schema = t.Table.Schema.O
			}
			a.ch <- MysqlOperationDDLTable{schema: schema, table: t.Table.Name.O, query: string(e.Query)}
		case *ast.CreateIndexStmt:
			schema := string(e.Schema)
			if len(schema) == 0 {
				schema = t.Table.Schema.O
			}
			a.ch <- MysqlOperationDDLTable{schema: schema, table: t.Table.Name.O, query: string(e.Query)}
		case *ast.DropIndexStmt:
		case *ast.CreateDatabaseStmt:
			a.ch <- OperationDDLDatabase{schema: t.Name.O, query: string(e.Query)}
		case *ast.AlterDatabaseStmt:
			a.ch <- OperationDDLDatabase{schema: t.Name.O, query: string(e.Query)}
		case *ast.DropDatabaseStmt:
			a.ch <- OperationDDLDatabase{schema: t.Name.O, query: string(e.Query)}
		case *ast.BeginStmt:
			fmt.Println("*ast.BeginStmt:", string(e.Query))
			a.ch <- MysqlOperationBegin{}
		case *ast.CommitStmt:
			// warning
		}
	}
	return nil
}

// mysql type -> golang type
// longtext -> []uint8
// enum int64
// mediumtext  []uint8
// test []uint8
// tinytext []uint8
// varchar string

// func GetMysqlDataTypeNameAndSqlColumn(tpDef string, colName string, tp byte, meta uint16) (string, SQL.NonAliasColumn) {
// 	// for unkown type, defaults to BytesColumn

// 	//get real string type
// 	if tp == mysql.MYSQL_TYPE_STRING {
// 		if meta >= 256 {
// 			b0 := uint8(meta >> 8)
// 			if b0&0x30 != 0x30 {
// 				tp = byte(b0 | 0x30)
// 			} else {
// 				tp = b0
// 			}
// 		}
// 	}
// 	//fmt.Println("column type:", colName, tp)
// 	switch tp {

// 	case mysql.MYSQL_TYPE_NULL:
// 		return C_unknownColType, SQL.BytesColumn(colName, SQL.NotNullable)
// 	case mysql.MYSQL_TYPE_LONG:
// 		return "int", SQL.IntColumn(colName, SQL.NotNullable)

// 	case mysql.MYSQL_TYPE_TINY:
// 		return "tinyint", SQL.IntColumn(colName, SQL.NotNullable)

// 	case mysql.MYSQL_TYPE_SHORT:
// 		return "smallint", SQL.IntColumn(colName, SQL.NotNullable)

// 	case mysql.MYSQL_TYPE_INT24:
// 		return "mediumint", SQL.IntColumn(colName, SQL.NotNullable)

// 	case mysql.MYSQL_TYPE_LONGLONG:
// 		return "bigint", SQL.IntColumn(colName, SQL.NotNullable)

// 	case mysql.MYSQL_TYPE_NEWDECIMAL:
// 		return "decimal", SQL.DoubleColumn(colName, SQL.NotNullable)

// 	case mysql.MYSQL_TYPE_FLOAT:
// 		return "float", SQL.DoubleColumn(colName, SQL.NotNullable)
// 	case mysql.MYSQL_TYPE_DOUBLE:
// 		return "double", SQL.DoubleColumn(colName, SQL.NotNullable)
// 	case mysql.MYSQL_TYPE_BIT:
// 		return "bit", SQL.IntColumn(colName, SQL.NotNullable)
// 	case mysql.MYSQL_TYPE_TIMESTAMP:
// 		//return "timestamp", SQL.DateTimeColumn(colName, SQL.NotNullable)
// 		return "timestamp", SQL.StrColumn(colName, SQL.UTF8, SQL.UTF8CaseInsensitive, SQL.NotNullable)
// 	case mysql.MYSQL_TYPE_TIMESTAMP2:
// 		//return "timestamp", SQL.DateTimeColumn(colName, SQL.NotNullable)
// 		return "timestamp", SQL.StrColumn(colName, SQL.UTF8, SQL.UTF8CaseInsensitive, SQL.NotNullable)
// 	case mysql.MYSQL_TYPE_DATETIME:
// 		//return "datetime", SQL.DateTimeColumn(colName, SQL.NotNullable)
// 		return "timestamp", SQL.StrColumn(colName, SQL.UTF8, SQL.UTF8CaseInsensitive, SQL.NotNullable)
// 	case mysql.MYSQL_TYPE_DATETIME2:
// 		//return "datetime", SQL.DateTimeColumn(colName, SQL.NotNullable)
// 		return "timestamp", SQL.StrColumn(colName, SQL.UTF8, SQL.UTF8CaseInsensitive, SQL.NotNullable)
// 	case mysql.MYSQL_TYPE_TIME:
// 		return "time", SQL.StrColumn(colName, SQL.UTF8, SQL.UTF8CaseInsensitive, SQL.NotNullable)
// 	case mysql.MYSQL_TYPE_TIME2:
// 		return "time", SQL.StrColumn(colName, SQL.UTF8, SQL.UTF8CaseInsensitive, SQL.NotNullable)
// 	case mysql.MYSQL_TYPE_DATE:
// 		return "date", SQL.StrColumn(colName, SQL.UTF8, SQL.UTF8CaseInsensitive, SQL.NotNullable)

// 	case mysql.MYSQL_TYPE_YEAR:
// 		return "year", SQL.IntColumn(colName, SQL.NotNullable)
// 	case mysql.MYSQL_TYPE_ENUM:
// 		return "enum", SQL.IntColumn(colName, SQL.NotNullable)
// 	case mysql.MYSQL_TYPE_SET:
// 		return "set", SQL.IntColumn(colName, SQL.NotNullable)
// 	case mysql.MYSQL_TYPE_BLOB:
// 		//text is stored as blob
// 		if strings.Contains(strings.ToLower(tpDef), "text") {
// 			return "blob", SQL.StrColumn(colName, SQL.UTF8, SQL.UTF8CaseInsensitive, SQL.NotNullable)
// 		}
// 		return "blob", SQL.BytesColumn(colName, SQL.NotNullable)
// 	case mysql.MYSQL_TYPE_VARCHAR,
// 		mysql.MYSQL_TYPE_VAR_STRING:

// 		return "varchar", SQL.StrColumn(colName, SQL.UTF8, SQL.UTF8CaseInsensitive, SQL.NotNullable)
// 	case mysql.MYSQL_TYPE_STRING:
// 		return "char", SQL.StrColumn(colName, SQL.UTF8, SQL.UTF8CaseInsensitive, SQL.NotNullable)
// 	// case mysql.MYSQL_TYPE_JSON:
// 	// 	//return "json", SQL.BytesColumn(colName, SQL.NotNullable)
// 	// 	return "json", SQL.StrColumn(colName, SQL.UTF8, SQL.UTF8CaseInsensitive, SQL.NotNullable)

// 	// case mysql.MYSQL_TYPE_GEOMETRY:
// 	// 	return "geometry", SQL.BytesColumn(colName, SQL.NotNullable)
// 	default:
// 		return C_unknownColType, SQL.BytesColumn(colName, SQL.NotNullable)
// 	}
// }

func reset_col_val(colum_type byte, col_val interface{}) {

}

func printType(value interface{}) {
	switch v := value.(type) {
	case int:
		fmt.Printf("Integer: %d\n", v)
	case float64:
		fmt.Printf("Float64: %f\n", v)
	case string:
		fmt.Printf("String: %s\n", v)
	case bool:
		fmt.Printf("Boolean: %t\n", v)
	default:
		fmt.Printf("Unsupported type: %T\n", v)
	}
}

// func BuildDMLInsertQuery(databaseName, tableName string, col_vals []interface{}, tm *replication.TableMapEvent) error {
// 	// ot := MysqlOperationDMLInsert{Database: string(tm.Schema), Table: string(tm.Table), Columns: []OperationTableColumn{}, PrimaryKey: tm.PrimaryKey}

// 	// for i := 0; i < int(tm.ColumnCount); i++ {
// 	// 	ot.Columns = append(ot.Columns, OperationTableColumn{ColumnName: string(tm.ColumnName[i]), ColumnType: tm.ColumnType[i], ColumnValue: col_vals[i]})
// 	// }
// 	// return ot, nil
// 	return nil
// }

// func BuildDMLUpdateQuery(databaseName, tableName string, rows [][]interface{}) error {
// 	return nil
// }

// func BuildDMLDeleteQuery(databaseName, tableName string, col_vals []interface{}, tm *replication.TableMapEvent) error {
// 	// ot := MysqlOperationDMLDelete{Database: string(tm.Schema), Table: string(tm.Table), Columns: []OperationTableColumn{}, PrimaryKey: tm.PrimaryKey}

// 	// for i := 0; i < int(tm.ColumnCount); i++ {
// 	// 	ot.Columns = append(ot.Columns, OperationTableColumn{ColumnName: string(tm.ColumnName[i]), ColumnType: tm.ColumnType[i], ColumnValue: col_vals[i]})
// 	// }
// 	// return ot, nil
// 	return nil

// }

func main() {
	cfg := replication.BinlogSyncerConfig{
		ServerID:        100,
		Flavor:          "mysql",
		Host:            "db1",
		Port:            3306,
		User:            "root",
		Password:        "root_password",
		Charset:         "utf8mb4",
		HeartbeatPeriod: time.Second * 10,
	}
	ch := make(chan MysqlOperation)

	abc := Canal{cfg: cfg, ch: ch}
	db, err := sql.Open("mysql", "root:root_password@tcp(db2:3306)/sys")
	if err != nil {
		fmt.Println(err)
		// panic(err)
	}
	m := &MysqlClient{DB: db, DmlCount: 0}
	abc.AddDestination(&MysqlDestination{ch: ch, Name: "mysql1", Client: m})
	// abc.AddDispatcher(MysqlDestination{ch: ch, Name: "mysql2"})
	println("start mysql sync")

	abc.Run()

	// println("start prometheus")
	// http.Handle("/metrics", promhttp.Handler())
	// http.ListenAndServe(":2112", nil)
}
