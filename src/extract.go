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
	logger             *Logger
	binlogSyncer       *replication.BinlogSyncer
	binlogSyncerConfig replication.BinlogSyncerConfig

	moCh      chan<- MysqlOperation
	gtidsetCh <-chan string

	metricCh chan<- interface{}

	ccc int
}

func NewBinlogExtract(logLevel int, config ReplicationConfig, moCh chan<- MysqlOperation, gtidsetCh <-chan string, metricCh chan<- interface{}) *BinlogExtract {
	cfg := replication.BinlogSyncerConfig{
		ServerID:        uint32(config.ServerID),
		Flavor:          "mysql",
		Host:            config.Host,
		Port:            uint16(config.Port),
		User:            config.User,
		Password:        config.Password,
		Charset:         "utf8mb4",
		HeartbeatPeriod: time.Millisecond * 500,
		ReadTimeout:     time.Minute * 10,
	}

	return &BinlogExtract{
		logger:             NewLogger(config.LogLevel, "extract"),
		binlogSyncer:       nil,
		binlogSyncerConfig: cfg,
		moCh:               moCh,
		gtidsetCh:          gtidsetCh,

		metricCh: metricCh,
		ccc:      0,
	}

}
func (bext *BinlogExtract) toMoCh(mo MysqlOperation) {
	bext.moCh <- mo
	bext.metricCh <- MetricUnit{Name: MetricExtractOperations, Value: 1}
}

func (bext *BinlogExtract) start(ctx context.Context, gtidset string) error {
	if bext.binlogSyncer == nil {
		bext.binlogSyncer = replication.NewBinlogSyncer(bext.binlogSyncerConfig)
	}

	gtidSet, err := mysql.ParseGTIDSet("mysql", gtidset)
	if err != nil {
		bext.logger.Error("ParseGTIDSet " + err.Error())
		return err
	}

	bext.logger.Info("start from gtidset:" + gtidSet.String())

	streamer, err := bext.binlogSyncer.StartSyncGTID(gtidSet)

	if err != nil {
		bext.logger.Error("start sync err:" + err.Error())
		return err
	}

	for {
		ev, err := streamer.GetEvent(ctx)

		if err != nil {
			if err == context.DeadlineExceeded {
				fmt.Println("Event fetch timed out:", err)
				return nil
			} else if err == context.Canceled {
				bext.binlogSyncer.Close()
				fmt.Println("Event handling canceled:", err)
				return nil
			}
			// fmt.Println("Error getting event:", err)
			bext.logger.Error("error event " + err.Error())
			return err
		}

		switch e := ev.Event.(type) {
		case *replication.RowsEvent:
			switch ev.Header.EventType {
			case replication.WRITE_ROWS_EVENTv0, replication.WRITE_ROWS_EVENTv1, replication.WRITE_ROWS_EVENTv2:
				if err := bext.handleEventWriteRows(e); err != nil {
					return err
				}
			case replication.UPDATE_ROWS_EVENTv0, replication.UPDATE_ROWS_EVENTv1, replication.UPDATE_ROWS_EVENTv2:
				if err := bext.handleEventUpdateRows(e); err != nil {
					return err
				}
			case replication.DELETE_ROWS_EVENTv0, replication.DELETE_ROWS_EVENTv1, replication.DELETE_ROWS_EVENTv2:
				if err := bext.handleEventDeleteRows(e); err != nil {
					return err
				}
			}
		case *replication.TableMapEvent:
		case *replication.GTIDEvent:
			if err := bext.handleEventGtid(e, ev.Header); err != nil {
				return err
			}
		case *replication.QueryEvent:
			if err := bext.handleQueryEvent(e, ev.Header); err != nil {
				return err
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

func (bext *BinlogExtract) handleEventWriteRows(e *replication.RowsEvent) error {
	for _, row := range e.Rows {
		mod := MysqlOperationDMLInsert{Database: string(e.Table.Schema), Table: string(e.Table.Table), Columns: []MysqlOperationDMLColumn{}, PrimaryKey: e.Table.PrimaryKey}
		for i := 0; i < int(e.Table.ColumnCount); i++ {
			mod.Columns = append(mod.Columns, MysqlOperationDMLColumn{ColumnName: string(e.Table.ColumnName[i]), ColumnType: e.Table.ColumnType[i], ColumnValue: row[i]})
		}
		bext.toMoCh(mod)
		bext.metricCh <- MetricUnit{Name: MetricReplDMLInsertTimes, Value: 1}
	}
	return nil
}

func (bext *BinlogExtract) handleEventDeleteRows(e *replication.RowsEvent) error {
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
		bext.toMoCh(mod)
		bext.metricCh <- MetricUnit{Name: MetricReplDMLDeleteTimes, Value: 1}
	}
	return nil
}

func (bext *BinlogExtract) handleEventUpdateRows(e *replication.RowsEvent) error {
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
		bext.toMoCh(mod)

		bext.metricCh <- MetricUnit{Name: MetricReplDMLUpdateTimes, Value: 1}
	}
	return nil
}

func (bext *BinlogExtract) handleEventGtid(e *replication.GTIDEvent, eh *replication.EventHeader) error {
	if gtidNext, err := e.GTIDNext(); err != nil {
		fmt.Println("Error retrieving GTID:", err)
		return err
	} else {
		// fmt.Println("GTID:", gtidNext.String())

		parts := strings.Split(gtidNext.String(), ":")
		// repl.logger.Debug("gtid " + gtidNext.String())
		if len(parts) == 2 {
			xid, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				fmt.Println("Error converting string to integer:", err)
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
			bext.toMoCh(MysqlOperationDDLTable{Schema: schema, Table: t.Table.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
		case *ast.DropTableStmt:
			schema := string(e.Schema)
			for _, tab := range t.Tables {
				if len(schema) == 0 {
					schema = tab.Schema.O
				}
				bext.toMoCh(MysqlOperationDDLTable{Schema: schema, Table: tab.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
			}
		case *ast.CreateTableStmt:
			schema := string(e.Schema)
			if len(schema) == 0 {
				schema = t.Table.Schema.O
			}
			bext.toMoCh(MysqlOperationDDLTable{Schema: schema, Table: t.Table.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
		case *ast.TruncateTableStmt:
			schema := string(e.Schema)
			if len(schema) == 0 {
				schema = t.Table.Schema.O
			}
			bext.toMoCh(MysqlOperationDDLTable{Schema: schema, Table: t.Table.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
		case *ast.CreateIndexStmt:
			schema := string(e.Schema)
			if len(schema) == 0 {
				schema = t.Table.Schema.O
			}
			bext.toMoCh(MysqlOperationDDLTable{Schema: schema, Table: t.Table.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
		case *ast.DropIndexStmt:
		case *ast.CreateDatabaseStmt:
			bext.toMoCh(MysqlOperationDDLDatabase{Schema: t.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
		case *ast.AlterDatabaseStmt:
			bext.toMoCh(MysqlOperationDDLDatabase{Schema: t.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
		case *ast.DropDatabaseStmt:
			bext.toMoCh(MysqlOperationDDLDatabase{Schema: t.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
		case *ast.BeginStmt:
			bext.toMoCh(MysqlOperationBegin{})
		case *ast.CommitStmt:
			// warning
		}
	}
	return nil
}
