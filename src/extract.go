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
	BinlogSyncerConfig replication.BinlogSyncerConfig
	moCh               chan<- MysqlOperation
	metricCh           chan<- MetricUnit
	StartSyncGtidsets  string
}

func NewBinlogExtract(logLevel int, config ReplicationConfig, startSyncGtidsets string, moCh chan<- MysqlOperation, metricCh chan<- MetricUnit) *BinlogExtract {
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
		EventCacheCount: 10,
	}

	return &BinlogExtract{
		Logger:             NewLogger(logLevel, "extract"),
		BinlogSyncerConfig: cfg,
		moCh:               moCh,
		metricCh:           metricCh,
		StartSyncGtidsets:  startSyncGtidsets,
	}
}

func (bext *BinlogExtract) toMoCh(mo MysqlOperation) {
	// bext.Logger.Debug("mo -> moCh ...")
	bext.moCh <- mo
	// bext.Logger.Debug("mo -> moCh ok.")
	bext.metricCh <- MetricUnit{Name: MetricReplExtractorTimestamp, Value: uint(mo.GetTimestamp())}
}

func (bext *BinlogExtract) Start(ctx context.Context) {
	bext.Logger.Info("Started.")
	defer bext.Logger.Info("Closed.")

	gtidSet, err := mysql.ParseGTIDSet("mysql", bext.StartSyncGtidsets)
	if err != nil {
		bext.Logger.Error("ParseGTIDSet: %s.", err)
		return
	}

	binlogSyncer := replication.NewBinlogSyncer(bext.BinlogSyncerConfig)
	bext.Logger.Info("binlogSyncer ready.")

	streamer, err := binlogSyncer.StartSyncGTID(gtidSet)
	if err != nil {
		bext.Logger.Error("Start streamer: %s.", err)
		return
	}
	bext.Logger.Info("Start streamer from gtidsets: '%s'.", gtidSet.String())

	binlogfile := ""

	for {
		ev, err := streamer.GetEvent(ctx)

		if err != nil {
			binlogSyncer.Close()
			if err == context.DeadlineExceeded {
				bext.Logger.Error("Event fetch timed out: %s.", err)
				return
			} else if err == context.Canceled {
				return
			}
			bext.Logger.Error("Event %s.", err)
			return
		}

		switch e := ev.Event.(type) {
		case *replication.RowsEvent:
			bext.Logger.Debug("Operation[binlogpos], event: %s, file: %s, pos: %d", ev.Header.EventType.String(), binlogfile, ev.Header.LogPos)

			switch ev.Header.EventType {
			case replication.WRITE_ROWS_EVENTv0, replication.WRITE_ROWS_EVENTv1, replication.WRITE_ROWS_EVENTv2:
				bext.handleEventWriteRows(e, ev.Header)
			case replication.UPDATE_ROWS_EVENTv0, replication.UPDATE_ROWS_EVENTv1, replication.UPDATE_ROWS_EVENTv2:
				bext.handleEventUpdateRows(e, ev.Header)
			case replication.DELETE_ROWS_EVENTv0, replication.DELETE_ROWS_EVENTv1, replication.DELETE_ROWS_EVENTv2:
				bext.handleEventDeleteRows(e, ev.Header)
			}
		case *replication.GTIDEvent:
			bext.Logger.Debug("Operation[binlogpos], event: %s, file: %s, pos: %d", ev.Header.EventType.String(), binlogfile, ev.Header.LogPos)

			if err := bext.handleEventGtid(e, ev.Header); err != nil {
				bext.Logger.Error("event: %s.", err)
				return
			}
		case *replication.QueryEvent:
			bext.Logger.Debug("Operation[binlogpos], event: %s, file: %s, pos: %d", ev.Header.EventType.String(), binlogfile, ev.Header.LogPos)

			// Used for checkpoint binlogpos
			bext.toMoCh(MysqlOperationBinLogPos{binlogfile, ev.Header.LogPos, ev.Header.Timestamp})
			bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationBinLogPos, Value: 1}

			if err := bext.handleQueryEvent(e, ev.Header); err != nil {
				bext.Logger.Error("event: %s.", err)
				return
			}
		case *replication.XIDEvent:
			bext.Logger.Debug("Operation[binlogpos], event: %s, file: %s, pos: %d", ev.Header.EventType.String(), binlogfile, ev.Header.LogPos)

			// Used for checkpoint binlogpos
			bext.toMoCh(MysqlOperationBinLogPos{binlogfile, ev.Header.LogPos, ev.Header.Timestamp})
			bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationBinLogPos, Value: 1}

			bext.toMoCh(MysqlOperationXid{ev.Header.Timestamp})
			bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationXid, Value: 1}
			bext.Logger.Debug("Operation[Xid]")
		case *replication.RotateEvent:
			switch ev.Header.EventType {
			case replication.ROTATE_EVENT:
				// if ev.Header.Timestamp >= 1 {
				binlogfile = string(e.NextLogName)
				// }
			default:
				bext.Logger.Warning("other RotateEvent", e)
			}
		case *replication.GenericEvent:
			switch ev.Header.EventType {
			case replication.HEARTBEAT_EVENT:
				bext.toMoCh(MysqlOperationHeartbeat{uint32(time.Now().Unix())})
				bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationHeartbeat, Value: 1}
			}
			// default:
			// bext.Logger.Debug("unprocess envet %s", ev.Header.EventType.String())
		}
	}
}

func (bext *BinlogExtract) handleEventWriteRows(e *replication.RowsEvent, eh *replication.EventHeader) {
	for _, row := range e.Rows {
		mod := MysqlOperationDMLInsert{
			Database:   string(e.Table.Schema),
			Table:      string(e.Table.Table),
			Columns:    []MysqlOperationDMLColumn{},
			PrimaryKey: e.Table.PrimaryKey,
			Timestamp:  eh.Timestamp,
		}
		for i := 0; i < int(e.Table.ColumnCount); i++ {
			modmlc := MysqlOperationDMLColumn{
				ColumnName:       string(e.Table.ColumnName[i]),
				ColumnType:       e.Table.ColumnType[i],
				ColumnValue:      row[i],
				ColumnValueIsNil: row[i] == nil,
			}
			mod.Columns = append(mod.Columns, modmlc)
		}
		bext.toMoCh(mod)
		bext.metricCh <- MetricUnit{Name: MetricReplDMLInsert, Value: 1, LabelPair: map[string]string{"database": string(e.Table.Schema), "table": string(e.Table.Table)}}
		bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationDMLInsert, Value: 1}
		bext.Logger.Debug("Operation[dmlinsert], SchemaContext: %s, Table: %s", string(e.Table.Schema), string(e.Table.Table))
	}
}

func (bext *BinlogExtract) handleEventDeleteRows(e *replication.RowsEvent, eh *replication.EventHeader) {
	for _, row := range e.Rows {
		mod := MysqlOperationDMLDelete{
			Database:   string(e.Table.Schema),
			Table:      string(e.Table.Table),
			Columns:    []MysqlOperationDMLColumn{},
			PrimaryKey: e.Table.PrimaryKey,
			Timestamp:  eh.Timestamp,
		}
		for i := 0; i < int(e.Table.ColumnCount); i++ {
			mod.Columns = append(mod.Columns, MysqlOperationDMLColumn{ColumnName: string(e.Table.ColumnName[i]), ColumnType: e.Table.ColumnType[i], ColumnValue: row[i], ColumnValueIsNil: row[i] == nil})
		}
		bext.toMoCh(mod)
		bext.metricCh <- MetricUnit{Name: MetricReplDMLDelete, Value: 1, LabelPair: map[string]string{"database": string(e.Table.Schema), "table": string(e.Table.Table)}}
		bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationDMLDelete, Value: 1}
		bext.Logger.Debug("Operation[dmldelete], SchemaContext: %s, Table: %s", string(e.Table.Schema), string(e.Table.Table))
	}
}

func (bext *BinlogExtract) handleEventUpdateRows(e *replication.RowsEvent, eh *replication.EventHeader) {
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
			mod.BeforeColumns = append(mod.BeforeColumns, MysqlOperationDMLColumn{ColumnName: string(e.Table.ColumnName[i]), ColumnType: e.Table.ColumnType[i], ColumnValue: before_value[i], ColumnValueIsNil: before_value[i] == nil})
			mod.AfterColumns = append(mod.AfterColumns, MysqlOperationDMLColumn{ColumnName: string(e.Table.ColumnName[i]), ColumnType: e.Table.ColumnType[i], ColumnValue: after_value[i], ColumnValueIsNil: after_value[i] == nil})
		}
		bext.toMoCh(mod)
		bext.metricCh <- MetricUnit{Name: MetricReplDMLUpdate, Value: 1, LabelPair: map[string]string{"database": string(e.Table.Schema), "table": string(e.Table.Table)}}
		bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationDMLUpdate, Value: 1}
		bext.Logger.Debug("Operation[dmlupdate], SchemaContext: %s, Table: %s", string(e.Table.Schema), string(e.Table.Table))
	}
}

func (bext *BinlogExtract) handleEventGtid(e *replication.GTIDEvent, eh *replication.EventHeader) error {
	if gtidNext, err := e.GTIDNext(); err != nil {
		bext.Logger.Error("Retrieving GTID: %s.", err)
		return err
	} else {
		parts := strings.Split(gtidNext.String(), ":")
		if len(parts) == 2 {
			xid, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				bext.Logger.Error("Error converting string to integer: %s.", err)
				return err
			} else {
				bext.toMoCh(MysqlOperationGTID{e.LastCommitted, eh.ServerID, eh.Timestamp, parts[0], xid})
				bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationGtid, Value: 1}
				bext.Logger.Debug("Operation[gtid], gtid: %s:%d", parts[0], xid)
				return nil
			}
		}
		return fmt.Errorf("unkonw handleEventGtids %#v", parts)
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
				bext.metricCh <- MetricUnit{Name: MetricReplDDLTable, Value: 1, LabelPair: map[string]string{"database": oldSchema, "table": tab.OldTable.Name.O}}
				bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationDDLTable, Value: 1}
				bext.Logger.Debug("Operation[ddltable], DDL:RENAME, SchemaContext: %s, Table: %s", oldSchema, tab.OldTable.Name.O)
			}
		case *ast.AlterTableStmt:
			schema := string(e.Schema)
			if len(schema) == 0 {
				schema = t.Table.Schema.O
			}
			bext.toMoCh(MysqlOperationDDLTable{SchemaContext: string(e.Schema), Schema: schema, Table: t.Table.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
			bext.metricCh <- MetricUnit{Name: MetricReplDDLTable, Value: 1, LabelPair: map[string]string{"database": schema, "table": t.Table.Name.O}}
			bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationDDLTable, Value: 1}
			bext.Logger.Debug("Operation[ddltable], DDL:ALTER, SchemaContext: %s, Table: %s", schema, t.Table.Name.O)
		case *ast.DropTableStmt:
			schema := string(e.Schema)
			for _, tab := range t.Tables {
				if len(schema) == 0 {
					schema = tab.Schema.O
				}
				Query := fmt.Sprintf("DROP TABLE IF EXISTS `%s`.`%s`", schema, tab.Name.O)
				bext.toMoCh(MysqlOperationDDLTable{SchemaContext: string(e.Schema), Schema: schema, Table: tab.Name.O, Query: Query, Timestamp: eh.Timestamp})
				bext.metricCh <- MetricUnit{Name: MetricReplDDLTable, Value: 1, LabelPair: map[string]string{"database": schema, "table": tab.Name.O}}
				bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationDDLTable, Value: 1}
				bext.Logger.Debug("Operation[ddltable], DDL:DROPTABLE, SchemaContext: %s, Table: %s", schema, tab.Name.O)
			}
		case *ast.CreateTableStmt:
			schema := string(e.Schema)
			if len(schema) == 0 {
				schema = t.Table.Schema.O
			}
			bext.toMoCh(MysqlOperationDDLTable{SchemaContext: string(e.Schema), Schema: schema, Table: t.Table.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
			bext.metricCh <- MetricUnit{Name: MetricReplDDLTable, Value: 1, LabelPair: map[string]string{"database": schema, "table": t.Table.Name.O}}
			bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationDDLTable, Value: 1}
			bext.Logger.Debug("Operation[ddltable], DDL:CREATE, SchemaContext: %s, Table: %s", schema, t.Table.Name.O)
		case *ast.TruncateTableStmt:
			schema := string(e.Schema)
			if len(schema) == 0 {
				schema = t.Table.Schema.O
			}
			bext.toMoCh(MysqlOperationDDLTable{SchemaContext: string(e.Schema), Schema: schema, Table: t.Table.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
			bext.metricCh <- MetricUnit{Name: MetricReplDDLTable, Value: 1, LabelPair: map[string]string{"database": schema, "table": t.Table.Name.O}}
			bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationDDLTable, Value: 1}
			bext.Logger.Debug("Operation[ddltable], DDL:TRUNCATE, SchemaContext: %s, Table: %s", schema, t.Table.Name.O)
		case *ast.DropIndexStmt:
			schema := string(e.Schema)
			if len(schema) == 0 {
				schema = t.Table.Schema.O
			}
			bext.toMoCh(MysqlOperationDDLTable{SchemaContext: string(e.Schema), Schema: schema, Table: t.Table.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
			bext.metricCh <- MetricUnit{Name: MetricReplDDLTable, Value: 1, LabelPair: map[string]string{"database": schema, "table": t.Table.Name.O}}
			bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationDDLTable, Value: 1}
			bext.Logger.Debug("Operation[ddltable], DDL:DROPINDEX, SchemaContext: %s, Table: %s", schema, t.Table.Name.O)
		case *ast.CreateIndexStmt:
			schema := string(e.Schema)
			if len(schema) == 0 {
				schema = t.Table.Schema.O
			}
			bext.toMoCh(MysqlOperationDDLTable{SchemaContext: string(e.Schema), Schema: schema, Table: t.Table.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
			bext.metricCh <- MetricUnit{Name: MetricReplDDLTable, Value: 1, LabelPair: map[string]string{"database": schema, "table": t.Table.Name.O}}
			bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationDDLTable, Value: 1}
			bext.Logger.Debug("Operation[ddltable], DDL:CREATEINDEX, SchemaContext: %s, Table: %s", schema, t.Table.Name.O)
		case *ast.CreateDatabaseStmt:
			bext.toMoCh(MysqlOperationDDLDatabase{Schema: t.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
			bext.metricCh <- MetricUnit{Name: MetricReplDDLDatabase, Value: 1, LabelPair: map[string]string{"database": t.Name.O}}
			bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationDDLDatabase, Value: 1}
			bext.Logger.Debug("Operation[ddldatabase], DDL:CREATE, Database: %s", t.Name.O)
		case *ast.AlterDatabaseStmt:
			bext.toMoCh(MysqlOperationDDLDatabase{Schema: t.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
			bext.metricCh <- MetricUnit{Name: MetricReplDDLDatabase, Value: 1, LabelPair: map[string]string{"database": t.Name.O}}
			bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationDDLDatabase, Value: 1}
			bext.Logger.Debug("Operation[ddldatabase], DDL:ALTER, Database: %s", t.Name.O)
		case *ast.DropDatabaseStmt:
			bext.toMoCh(MysqlOperationDDLDatabase{Schema: t.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
			bext.metricCh <- MetricUnit{Name: MetricReplDDLDatabase, Value: 1, LabelPair: map[string]string{"database": t.Name.O}}
			bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationDDLDatabase, Value: 1}
			bext.Logger.Debug("Operation[ddldatabase], DDL:DROP, Database: %s", t.Name.O)
		case *ast.BeginStmt:
			bext.toMoCh(MysqlOperationBegin{Timestamp: eh.Timestamp})
			bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationBegin, Value: 1}
			bext.Logger.Debug("Operation[begin]")
		case *ast.CommitStmt:
			bext.Logger.Warning("event query: %s", string(e.Query))
		}
	}
	return nil
}
