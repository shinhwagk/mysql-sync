package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
	"github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"

	_ "github.com/pingcap/tidb/pkg/types/parser_driver"
)

func handleQueryEvent(e *replication.QueryEvent, eh *replication.EventHeader) error {
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
			oldSchema := string(e.Schema)
			newSchema := string(e.Schema)

			for _, tab := range t.TableToTables {
				if len(tab.OldTable.Schema.O) != 0 {
					oldSchema = tab.OldTable.Schema.O
				}

				if len(tab.NewTable.Schema.O) != 0 {
					newSchema = tab.NewTable.Schema.O
				}
				Query := fmt.Sprintf("RENAME TABLE `%s`.`%s` TO `%s`.`%s`", oldSchema, tab.OldTable.Name.O, newSchema, tab.NewTable.Name)
				fmt.Println("Quewry " + string(e.Schema) + " " + Query)
				bext.toMoCh(MysqlOperationDDLTable{Schema: string(e.Schema), Table: tab.OldTable.Name.O, Query: Query, Timestamp: eh.Timestamp})
			}
		case *ast.AlterTableStmt:
			schema := string(e.Schema)
			if len(schema) == 0 {
				schema = t.Table.Schema.O
			}

		case *ast.DropTableStmt:
			schema := string(e.Schema)
			for _, tab := range t.Tables {
				if len(schema) == 0 {
					schema = tab.Schema.O
				}
				Query := fmt.Sprintf("DROP TABLE `%s`.`%s`", schema, tab.Name.O)
				fmt.Println("Quewry", Query)

				// bext.toMoCh(MysqlOperationDDLTable{Schema: schema, Table: tab.Name.O, Query: Query, Timestamp: eh.Timestamp})
			}
		case *ast.CreateTableStmt:
			schema := string(e.Schema)
			if len(schema) == 0 {
				schema = t.Table.Schema.O
			}

		case *ast.TruncateTableStmt:
			schema := string(e.Schema)
			if len(schema) == 0 {
				schema = t.Table.Schema.O
			}

		case *ast.CreateIndexStmt:
			schema := string(e.Schema)
			if len(schema) == 0 {
				schema = t.Table.Schema.O
			}

		case *ast.DropIndexStmt:
		case *ast.CreateDatabaseStmt:

		case *ast.AlterDatabaseStmt:

		case *ast.DropDatabaseStmt:

		case *ast.BeginStmt:
		case *ast.CommitStmt:
			// warning
		}
	}
	return nil
}

func main() {
	// 假设这是配置和创建 BinlogSyncer 的代码段
	config := replication.BinlogSyncerConfig{
		ServerID: 1001,
		Flavor:   "mysql",
		Host:     "db1",
		Port:     3306,
		User:     "root",
		Password: "root_password",
	}

	syncer := replication.NewBinlogSyncer(config)
	streamer, err := syncer.StartSync(mysql.Position{"mysql-bin.000001", 0})

	if err != nil {
		log.Fatal("Failed to start sync:", err)
	}

	if streamer == nil {
		log.Fatal("Streamer is nil")
	}

	var xxx []byte = make([]byte, 10)

	fmt.Println("[]bye '" + string(xxx) + "'")

	go func() {
		time.Sleep(time.Second * 4)
		// binlogExtract.Stop()
		childCancel()
		fmt.Println("binlogext stop")
	}()

	for {
		ev, err := streamer.GetEvent(context.Background())
		if err != nil {
			log.Fatal("Error getting event:", err)
			break
		}

		switch e := ev.Event.(type) {
		case *replication.QueryEvent:
			handleQueryEvent(e, ev.Header)
			// fmt.Println(string(e.Query))
		}

		// 处理 binlog 事件
		// log.Printf("Event: %v", ev.Header.EventType)
	}
}
