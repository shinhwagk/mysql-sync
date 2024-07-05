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

	_ "github.com/pingcap/tidb/pkg/types/parser_driver"
)

type BinlogExtract struct {
	Logger             *Logger
	binlogSyncer       *replication.BinlogSyncer
	binlogSyncerConfig replication.BinlogSyncerConfig

	moCh chan<- MysqlOperation

	metricCh chan<- MetricUnit
}

func NewBinlogExtract(logLevel int, config ReplicationConfig, moCh chan<- MysqlOperation, metricCh chan<- MetricUnit) *BinlogExtract {
	cfg := replication.BinlogSyncerConfig{
		ServerID:        uint32(config.ServerID),
		Flavor:          "mysql",
		Host:            config.Host,
		Port:            uint16(config.Port),
		User:            config.User,
		Password:        config.Password,
		Charset:         "utf8mb4",
		HeartbeatPeriod: time.Millisecond * 100,
		ReadTimeout:     time.Minute * 10,
	}

	return &BinlogExtract{
		Logger:             NewLogger(config.LogLevel, "extract"),
		binlogSyncer:       nil,
		binlogSyncerConfig: cfg,
		moCh:               moCh,
		metricCh:           metricCh,
	}
}

func (bext *BinlogExtract) toMoCh(mo MysqlOperation) {
	// bext.Logger.Debug("mo -> moCh ...")
	bext.moCh <- mo
	// bext.Logger.Debug("mo -> moCh ok.")
	bext.metricCh <- MetricUnit{Name: MetricExtractOperations, Value: 1}
	bext.metricCh <- MetricUnit{Name: MetricReplDelay, Value: uint(time.Now().Unix() - int64(mo.GetTimestamp()))}
}

func (bext *BinlogExtract) Start(ctx context.Context, gtidsets string) {
	bext.Logger.Info("Started.")
	defer bext.Logger.Info("Closed.")

	if bext.binlogSyncer == nil {
		bext.binlogSyncer = replication.NewBinlogSyncer(bext.binlogSyncerConfig)
	}

	gtidSet, err := mysql.ParseGTIDSet("mysql", gtidsets)
	if err != nil {
		bext.Logger.Error("ParseGTIDSet " + err.Error())
		return
	}

	bext.Logger.Info("Start from gtidsets:" + gtidSet.String())

	streamer, err := bext.binlogSyncer.StartSyncGTID(gtidSet)
	if err != nil {
		bext.Logger.Error("start sync err:" + err.Error())
		return
	}

	bext.Logger.Info("binlogSyncer ready.")

	for {
		ev, err := streamer.GetEvent(ctx)

		if err != nil {
			bext.binlogSyncer.Close()
			bext.binlogSyncer = nil
			if err == context.DeadlineExceeded {
				bext.Logger.Error("Event fetch timed out: %s", err)
				return
			} else if err == context.Canceled {
				bext.Logger.Error("Event handling canceled: %s", err)
				return
			}
			bext.Logger.Error("error event " + err.Error())
			return
		}

		switch e := ev.Event.(type) {
		case *replication.RowsEvent:
			switch ev.Header.EventType {
			case replication.WRITE_ROWS_EVENTv0, replication.WRITE_ROWS_EVENTv1, replication.WRITE_ROWS_EVENTv2:
				if err := bext.handleEventWriteRows(e, ev.Header); err != nil {
					bext.Logger.Error("error event " + err.Error())
					return
				}
			case replication.UPDATE_ROWS_EVENTv0, replication.UPDATE_ROWS_EVENTv1, replication.UPDATE_ROWS_EVENTv2:
				if err := bext.handleEventUpdateRows(e, ev.Header); err != nil {
					bext.Logger.Error("error event " + err.Error())
					return
				}
			case replication.DELETE_ROWS_EVENTv0, replication.DELETE_ROWS_EVENTv1, replication.DELETE_ROWS_EVENTv2:
				if err := bext.handleEventDeleteRows(e, ev.Header); err != nil {
					bext.Logger.Error("error event " + err.Error())
					return
				}
			}
		case *replication.TableMapEvent:
		case *replication.GTIDEvent:
			if err := bext.handleEventGtid(e, ev.Header); err != nil {
				bext.Logger.Error("error event " + err.Error())
				return
			}
		case *replication.QueryEvent:
			if err := bext.handleQueryEvent(e, ev.Header); err != nil {
				bext.Logger.Error("error event " + err.Error())
				return
			}
		case *replication.XIDEvent:
			bext.toMoCh(MysqlOperationXid{ev.Header.Timestamp})
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
				bext.toMoCh(MysqlOperationHeartbeat{uint32(time.Now().Unix())})
			}
		}
	}
}

func (bext *BinlogExtract) handleEventWriteRows(e *replication.RowsEvent, eh *replication.EventHeader) error {
	for _, row := range e.Rows {
		mod := MysqlOperationDMLInsert{
			Database:   string(e.Table.Schema),
			Table:      string(e.Table.Table),
			Columns:    []MysqlOperationDMLColumn{},
			PrimaryKey: e.Table.PrimaryKey,
			Timestamp:  eh.Timestamp,
		}
		for i := 0; i < int(e.Table.ColumnCount); i++ {
			mod.Columns = append(mod.Columns, MysqlOperationDMLColumn{ColumnName: string(e.Table.ColumnName[i]), ColumnType: e.Table.ColumnType[i], ColumnValue: row[i]})
		}
		bext.toMoCh(mod)
		bext.metricCh <- MetricUnit{Name: MetricReplDMLInsertTimes, Value: 1}
	}
	return nil
}

func (bext *BinlogExtract) handleEventDeleteRows(e *replication.RowsEvent, eh *replication.EventHeader) error {
	for _, row := range e.Rows {
		mod := MysqlOperationDMLDelete{
			Database:   string(e.Table.Schema),
			Table:      string(e.Table.Table),
			Columns:    []MysqlOperationDMLColumn{},
			PrimaryKey: e.Table.PrimaryKey,
			Timestamp:  eh.Timestamp,
		}
		for i := 0; i < int(e.Table.ColumnCount); i++ {
			mod.Columns = append(mod.Columns, MysqlOperationDMLColumn{ColumnName: string(e.Table.ColumnName[i]), ColumnType: e.Table.ColumnType[i], ColumnValue: row[i]})
		}
		bext.toMoCh(mod)
		bext.metricCh <- MetricUnit{Name: MetricReplDMLDeleteTimes, Value: 1}
	}
	return nil
}

func (bext *BinlogExtract) handleEventUpdateRows(e *replication.RowsEvent, eh *replication.EventHeader) error {
	for i := 0; i < len(e.Rows); i += 2 {
		before_value := e.Rows[i]
		after_value := e.Rows[i+1]

		mod := MysqlOperationDMLUpdate{
			Database:      string(e.Table.Schema),
			Table:         string(e.Table.Table),
			BeforeColumns: []MysqlOperationDMLColumn{},
			AfterColumns:  []MysqlOperationDMLColumn{},
			PrimaryKey:    e.Table.PrimaryKey,
			Timestamp:     eh.Timestamp,
		}

		for i := 0; i < int(e.Table.ColumnCount); i++ {
			mod.BeforeColumns = append(mod.BeforeColumns, MysqlOperationDMLColumn{ColumnName: string(e.Table.ColumnName[i]), ColumnType: e.Table.ColumnType[i], ColumnValue: before_value[i]})
			mod.AfterColumns = append(mod.AfterColumns, MysqlOperationDMLColumn{ColumnName: string(e.Table.ColumnName[i]), ColumnType: e.Table.ColumnType[i], ColumnValue: after_value[i]})
		}
		bext.toMoCh(mod)
		bext.metricCh <- MetricUnit{Name: MetricReplDMLUpdateTimes, Value: 1}
	}
	return nil
}

func (bext *BinlogExtract) handleEventGtid(e *replication.GTIDEvent, eh *replication.EventHeader) error {
	if gtidNext, err := e.GTIDNext(); err != nil {
		bext.Logger.Error("Retrieving GTID: %s", err.Error())
		return err
	} else {
		parts := strings.Split(gtidNext.String(), ":")
		if len(parts) == 2 {
			xid, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				bext.Logger.Error("Error converting string to integer: %s", err.Error())
				return err
			} else {
				bext.toMoCh(MysqlOperationGTID{e.LastCommitted, eh.ServerID, eh.Timestamp, parts[0], xid})
				return nil
			}
		}
		return fmt.Errorf("unkonw handleEventGtids", parts)
	}
}
func (bext *BinlogExtract) handleQueryEvent(e *replication.QueryEvent, eh *replication.EventHeader) error {
	parser := parser.New()
	stmts, warns, err := parser.Parse(string(e.Query), "", "")
	for _, warn := range warns {
		bext.Logger.Warning(warn.Error())
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
				bext.toMoCh(MysqlOperationDDLTable{SchemaContext: string(e.Schema), Schema: oldSchema, Table: tab.OldTable.Name.O, Query: Query, Timestamp: eh.Timestamp})
			}
		case *ast.AlterTableStmt:
			schema := string(e.Schema)
			if len(schema) == 0 {
				schema = t.Table.Schema.O
			}
			bext.toMoCh(MysqlOperationDDLTable{SchemaContext: string(e.Schema), Schema: schema, Table: t.Table.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
		case *ast.DropTableStmt:
			schema := string(e.Schema)
			for _, tab := range t.Tables {
				if len(schema) == 0 {
					schema = tab.Schema.O
				}
				Query := fmt.Sprintf("DROP TABLE IF EXISTS `%s`.`%s`", schema, tab.Name.O)
				bext.toMoCh(MysqlOperationDDLTable{SchemaContext: string(e.Schema), Schema: schema, Table: tab.Name.O, Query: Query, Timestamp: eh.Timestamp})
			}
		case *ast.CreateTableStmt:
			schema := string(e.Schema)
			if len(schema) == 0 {
				schema = t.Table.Schema.O
			}
			bext.toMoCh(MysqlOperationDDLTable{SchemaContext: string(e.Schema), Schema: schema, Table: t.Table.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
		case *ast.TruncateTableStmt:
			schema := string(e.Schema)
			if len(schema) == 0 {
				schema = t.Table.Schema.O
			}
			bext.toMoCh(MysqlOperationDDLTable{SchemaContext: string(e.Schema), Schema: schema, Table: t.Table.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
		case *ast.DropIndexStmt:
		case *ast.CreateDatabaseStmt:
			bext.toMoCh(MysqlOperationDDLDatabase{Schema: t.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
		case *ast.AlterDatabaseStmt:
			bext.toMoCh(MysqlOperationDDLDatabase{Schema: t.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
		case *ast.DropDatabaseStmt:
			bext.toMoCh(MysqlOperationDDLDatabase{Schema: t.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
		case *ast.BeginStmt:
			bext.toMoCh(MysqlOperationBegin{Timestamp: eh.Timestamp})
		case *ast.CommitStmt:
			// warning
		}
	}
	return nil
}
