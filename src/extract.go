package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
	"github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"
)

type BinlogExtract struct {
	logger             *Logger
	binlogSyncer       *replication.BinlogSyncer
	binlogSyncerConfig replication.BinlogSyncerConfig

	running bool
}

func NewBinlogExtract(logLevel int, config ReplicationConfig) *BinlogExtract {
	cfg := replication.BinlogSyncerConfig{
		ServerID:        uint32(config.ServerID),
		Flavor:          "mysql",
		Host:            config.Host,
		Port:            uint16(config.Port),
		User:            config.User,
		Password:        config.Password,
		Charset:         "utf8mb4",
		HeartbeatPeriod: time.Second * 5,
	}

	return &BinlogExtract{
		logger:             NewLogger(config.LogLevel, "extract"),
		binlogSyncer:       nil,
		binlogSyncerConfig: cfg,
	}

}

// func (ext *BinlogExtract) RestartSync(ctx context.Context, gtid string, ch chan<- MysqlOperation) error {
// 	if ext.binlogSyncer != nil {
// 		ext.binlogSyncer.Close()
// 	}

// 	ext.binlogSyncer = replication.NewBinlogSyncer(ext.binlogSyncerConfig)

// 	gtidSet, err := mysql.ParseGTIDSet("mysql", gtid)
// 	if err != nil {
// 		ext.logger.Error("ParseGTIDSet " + err.Error())
// 		return err
// 	}

// 	ext.logger.Info("start")

// 	err = ext.start(ctx, gtidSet, ch)

// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

func (repl BinlogExtract) start(ctx context.Context, gtidset string, ch chan<- MysqlOperation) error {
	repl.running = true

	if repl.binlogSyncer == nil {
		repl.binlogSyncer = replication.NewBinlogSyncer(repl.binlogSyncerConfig)
	}

	gtidSet, err := mysql.ParseGTIDSet("mysql", gtidset)
	if err != nil {
		repl.logger.Error("ParseGTIDSet " + err.Error())
		return err
	}

	streamer, err := repl.binlogSyncer.StartSyncGTID(gtidSet)

	if err != nil {
		repl.logger.Error("start sync err:" + err.Error())
		return err
	}

	for {
		ev, err := streamer.GetEvent(ctx)

		if err != nil {
			if err == context.DeadlineExceeded {
				fmt.Println("Event fetch timed out:", err)
				return nil
			} else if err == context.Canceled {
				repl.binlogSyncer.Close()
				fmt.Println("Event handling canceled:", err)
				return nil
			}
			// fmt.Println("Error getting event:", err)
			repl.logger.Error("error event " + err.Error())
			return err
		}

		repl.logger.Debug("parse mysql event " + ev.Header.EventType.String())

		switch e := ev.Event.(type) {
		case *replication.RowsEvent:
			switch ev.Header.EventType {
			case replication.WRITE_ROWS_EVENTv0, replication.WRITE_ROWS_EVENTv1, replication.WRITE_ROWS_EVENTv2:
				if err := repl.handleEventWriteRows(e, ch); err != nil {
					return err
				}
			case replication.UPDATE_ROWS_EVENTv0, replication.UPDATE_ROWS_EVENTv1, replication.UPDATE_ROWS_EVENTv2:
				if err := repl.handleEventUpdateRows(e, ch); err != nil {
					return err
				}
			case replication.DELETE_ROWS_EVENTv0, replication.DELETE_ROWS_EVENTv1, replication.DELETE_ROWS_EVENTv2:
				if err := repl.handleEventDeleteRows(e, ch); err != nil {
					return err
				}
			}
		case *replication.TableMapEvent:
		case *replication.GTIDEvent:
			if err := repl.handleEventGtid(e, ev.Header, ch); err != nil {
				return err
			}
		case *replication.QueryEvent:
			if err := repl.handleQueryEvent(e, ev.Header, ch); err != nil {
				return err
			}
		case *replication.XIDEvent:
			ch <- MysqlOperationXid{ev.Header.Timestamp}
			repl.logger.Debug("MysqlOperationXid -> ch")
		case *replication.RotateEvent:
			switch ev.Header.EventType {
			case replication.ROTATE_EVENT:
				if ev.Header.Timestamp >= 1 {
					// repl.binlogfile = string(e.NextLogName)
				}
			default:
				fmt.Println("other RotateEvent", e)
			}
		case *replication.GenericEvent:
			switch ev.Header.EventType {
			case replication.HEARTBEAT_EVENT:
				ch <- MysqlOperationHeartbeat{uint32(time.Now().Unix())}
				repl.logger.Debug("MysqlOperationHeartbeat -> ch")
			}
		}
	}
}

func (repl BinlogExtract) handleEventWriteRows(e *replication.RowsEvent, ch chan<- MysqlOperation) error {
	for _, row := range e.Rows {
		mod := MysqlOperationDMLInsert{Database: string(e.Table.Schema), Table: string(e.Table.Table), Columns: []MysqlOperationDMLColumn{}, PrimaryKey: e.Table.PrimaryKey}
		for i := 0; i < int(e.Table.ColumnCount); i++ {
			mod.Columns = append(mod.Columns, MysqlOperationDMLColumn{ColumnName: string(e.Table.ColumnName[i]), ColumnType: e.Table.ColumnType[i], ColumnValue: row[i]})
		}
		ch <- mod
	}
	return nil
}

func (repl BinlogExtract) handleEventDeleteRows(e *replication.RowsEvent, ch chan<- MysqlOperation) error {
	for _, row := range e.Rows {
		mod := MysqlOperationDMLDelete{
			Database:   string(e.Table.Schema),
			Table:      string(e.Table.Table),
			Columns:    []MysqlOperationDMLColumn{},
			PrimaryKey: e.Table.PrimaryKey,
		}
		for i := 0; i < int(e.Table.ColumnCount); i++ {
			mod.Columns = append(mod.Columns, MysqlOperationDMLColumn{ColumnName: string(e.Table.ColumnName[i]), ColumnType: e.Table.ColumnType[i], ColumnValue: row[i]})
		}
		ch <- mod
	}
	return nil
}

func (ext BinlogExtract) handleEventUpdateRows(e *replication.RowsEvent, ch chan<- MysqlOperation) error {
	for i := 0; i < len(e.Rows); i += 2 {
		before_value := e.Rows[i]
		after_value := e.Rows[i+1]

		mod := MysqlOperationDMLUpdate{
			Database:      string(e.Table.Schema),
			Table:         string(e.Table.Table),
			BeforeColumns: []MysqlOperationDMLColumn{},
			AfterColumns:  []MysqlOperationDMLColumn{},
			PrimaryKey:    e.Table.PrimaryKey,
		}

		for i := 0; i < int(e.Table.ColumnCount); i++ {
			mod.BeforeColumns = append(mod.BeforeColumns, MysqlOperationDMLColumn{ColumnName: string(e.Table.ColumnName[i]), ColumnType: e.Table.ColumnType[i], ColumnValue: before_value[i]})
			mod.AfterColumns = append(mod.AfterColumns, MysqlOperationDMLColumn{ColumnName: string(e.Table.ColumnName[i]), ColumnType: e.Table.ColumnType[i], ColumnValue: after_value[i]})
		}
		ch <- mod
	}
	return nil
}

func (repl BinlogExtract) handleEventGtid(e *replication.GTIDEvent, eh *replication.EventHeader, ch chan<- MysqlOperation) error {
	if gtidNext, err := e.GTIDNext(); err != nil {
		fmt.Println("Error retrieving GTID:", err)
		return err
	} else {
		// fmt.Println("GTID:", gtidNext.String())

		parts := strings.Split(gtidNext.String(), ":")
		if len(parts) == 2 {
			xid, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				fmt.Println("Error converting string to integer:", err)
				return err
			} else {
				ch <- MysqlOperationGTID{e.LastCommitted, eh.ServerID, eh.Timestamp, parts[0], xid}
				return nil
			}
		}
		return fmt.Errorf("unkonw handleEventGtids", parts)
	}
}
func (repl BinlogExtract) handleQueryEvent(e *replication.QueryEvent, eh *replication.EventHeader, ch chan<- MysqlOperation) error {
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
			ch <- MysqlOperationDDLTable{Schema: schema, Table: t.Table.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp}
		case *ast.DropTableStmt:
			schema := string(e.Schema)
			for _, tab := range t.Tables {
				if len(schema) == 0 {
					schema = tab.Schema.O
				}
				ch <- MysqlOperationDDLTable{Schema: schema, Table: tab.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp}
			}
		case *ast.CreateTableStmt:
			schema := string(e.Schema)
			if len(schema) == 0 {
				schema = t.Table.Schema.O
			}
			ch <- MysqlOperationDDLTable{Schema: schema, Table: t.Table.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp}
		case *ast.TruncateTableStmt:
			schema := string(e.Schema)
			if len(schema) == 0 {
				schema = t.Table.Schema.O
			}
			ch <- MysqlOperationDDLTable{Schema: schema, Table: t.Table.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp}
		case *ast.CreateIndexStmt:
			schema := string(e.Schema)
			if len(schema) == 0 {
				schema = t.Table.Schema.O
			}
			ch <- MysqlOperationDDLTable{Schema: schema, Table: t.Table.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp}
		case *ast.DropIndexStmt:
		case *ast.CreateDatabaseStmt:
			ch <- MysqlOperationDDLDatabase{Schema: t.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp}
		case *ast.AlterDatabaseStmt:
			ch <- MysqlOperationDDLDatabase{Schema: t.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp}
		case *ast.DropDatabaseStmt:
			ch <- MysqlOperationDDLDatabase{Schema: t.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp}
		case *ast.BeginStmt:
			ch <- MysqlOperationBegin{}
		case *ast.CommitStmt:
			// warning
		}
	}
	return nil
}
