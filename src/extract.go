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
	BinlogFile         string
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
		BinlogFile:         "",
	}
}

func (bext *BinlogExtract) toMoCh(mo MysqlOperation) {
	// bext.Logger.Debug("mo -> moCh ...")
	bext.moCh <- mo
	// bext.Logger.Debug("mo -> moCh ok.")
	bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperations, Value: 1}
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

		moblp := MysqlOperationBinLogPos{ev.Header.ServerID, ev.Header.EventType.String(), bext.BinlogFile, ev.Header.LogPos, ev.Header.Timestamp}

		switch e := ev.Event.(type) {
		case *replication.RowsEvent:
			bext.Logger.Debug("Operation[binlogpos] -- server id: %d, event: %s, file: %s, pos: %d", moblp.ServerID, moblp.Event, moblp.File, moblp.Pos)

			bext.toMoCh(moblp)

			switch ev.Header.EventType {
			case replication.WRITE_ROWS_EVENTv0, replication.WRITE_ROWS_EVENTv1, replication.WRITE_ROWS_EVENTv2:
				bext.handleEventWriteRows(e, ev.Header)
			case replication.UPDATE_ROWS_EVENTv0, replication.UPDATE_ROWS_EVENTv1, replication.UPDATE_ROWS_EVENTv2:
				bext.handleEventUpdateRows(e, ev.Header)
			case replication.DELETE_ROWS_EVENTv0, replication.DELETE_ROWS_EVENTv1, replication.DELETE_ROWS_EVENTv2:
				bext.handleEventDeleteRows(e, ev.Header)
			}
		case *replication.GTIDEvent:
			bext.Logger.Debug("Operation[binlogpos] -- server id: %d, event: %s, file: %s, pos: %d", moblp.ServerID, moblp.Event, moblp.File, moblp.Pos)

			bext.toMoCh(moblp)

			if err := bext.handleEventGtid(e, ev.Header); err != nil {
				bext.Logger.Error("event: %s.", err)
				return
			}
		case *replication.QueryEvent:
			bext.Logger.Debug("Operation[binlogpos] -- server id: %d, event: %s, file: %s, pos: %d", moblp.ServerID, moblp.Event, moblp.File, moblp.Pos)

			bext.toMoCh(moblp)

			if err := bext.handleQueryEvent(e, ev.Header); err != nil {
				bext.Logger.Error("event: %s.", err)
				return
			}
		case *replication.XIDEvent:
			bext.Logger.Debug("Operation[binlogpos] -- server id: %d, event: %s, file: %s, pos: %d", moblp.ServerID, moblp.Event, moblp.File, moblp.Pos)

			bext.toMoCh(moblp)

			bext.toMoCh(MysqlOperationXid{ev.Header.Timestamp})
			bext.Logger.Debug("Operation[Xid]")
		case *replication.RotateEvent:
			bext.Logger.Debug("Operation[binlogpos] -- server id: %d, event: %s, file: %s, pos: %d", moblp.ServerID, moblp.Event, moblp.File, moblp.Pos)

			bext.toMoCh(moblp)

			switch ev.Header.EventType {
			case replication.ROTATE_EVENT:
				// if ev.Header.Timestamp >= 1 {
				bext.BinlogFile = string(e.NextLogName)
				// }
			default:
				bext.Logger.Warning("other RotateEvent")
			}
		case *replication.GenericEvent:
			switch ev.Header.EventType {
			case replication.HEARTBEAT_EVENT:
				bext.toMoCh(MysqlOperationHeartbeat{uint32(time.Now().Unix())})
				bext.Logger.Trace("Operation[Heartbeat]")
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
		bext.Logger.Debug("Operation[dmlinsert] -- SchemaContext: %s, Table: %s", string(e.Table.Schema), string(e.Table.Table))
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
		bext.Logger.Debug("Operation[dmlupdate] -- SchemaContext: %s, Table: %s", string(e.Table.Schema), string(e.Table.Table))
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
				bext.Logger.Debug("Operation[gtid] -- gtid: %s:%d, lastcommitted: %d", parts[0], xid, e.LastCommitted)
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
			schemaContext := string(e.Schema)

			for _, tab := range t.TableToTables {
				oldDatabase := tab.OldTable.Schema.O
				if len(oldDatabase) == 0 {
					oldDatabase = schemaContext
				}

				newDatabase := tab.NewTable.Schema.O
				if len(newDatabase) == 0 {
					newDatabase = schemaContext
				}

				query := fmt.Sprintf("RENAME TABLE `%s`.`%s` TO `%s`.`%s`", oldDatabase, tab.OldTable.Name.O, newDatabase, tab.NewTable.Name)
				bext.toMoCh(MysqlOperationDDLTable{SchemaContext: schemaContext, Database: oldDatabase, Table: tab.OldTable.Name.O, Query: query, Timestamp: eh.Timestamp})
				bext.metricCh <- MetricUnit{Name: MetricReplDDLTable, Value: 1, LabelPair: map[string]string{"database": oldDatabase, "table": tab.OldTable.Name.O}}
				bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationDDLTable, Value: 1}
				bext.Logger.Debug("Operation[ddltable], DDL:TableRename, SchemaContext: %s, Database: %s, Table: %s, Query: %s", schemaContext, oldDatabase, tab.OldTable.Name.O, query)
			}
		case *ast.AlterTableStmt:
			schemaContext := string(e.Schema)

			database := t.Table.Schema.O
			if len(database) == 0 {
				database = schemaContext
			}

			bext.toMoCh(MysqlOperationDDLTable{SchemaContext: schemaContext, Database: database, Table: t.Table.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
			bext.metricCh <- MetricUnit{Name: MetricReplDDLTable, Value: 1, LabelPair: map[string]string{"database": schemaContext, "table": t.Table.Name.O}}
			bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationDDLTable, Value: 1}
			bext.Logger.Debug("Operation[ddltable], DDL:TableCreate, SchemaContext: `%s`, Database: `%s`, Table: `%s`, Query: '%s'", schemaContext, database, t.Table.Name.O, string(e.Query))
		case *ast.DropTableStmt:
			schemaContext := string(e.Schema)

			for _, tab := range t.Tables {
				database := tab.Schema.O
				if len(database) == 0 {
					database = schemaContext
				}

				query := fmt.Sprintf("DROP TABLE IF EXISTS `%s`.`%s`", database, tab.Name.O)
				bext.toMoCh(MysqlOperationDDLTable{SchemaContext: schemaContext, Database: database, Table: tab.Name.O, Query: query, Timestamp: eh.Timestamp})
				bext.metricCh <- MetricUnit{Name: MetricReplDDLTable, Value: 1, LabelPair: map[string]string{"database": database, "table": tab.Name.O}}
				bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationDDLTable, Value: 1}
				bext.Logger.Debug("Operation[ddltable], DDL:TableDrop, SchemaContext: %s, Database: %s, Table: %s, Query: %s", schemaContext, database, tab.Name.O, query)
			}
		case *ast.CreateTableStmt:
			schemaContext := string(e.Schema)

			database := t.Table.Schema.O
			if len(database) == 0 {
				database = schemaContext
			}

			bext.toMoCh(MysqlOperationDDLTable{SchemaContext: schemaContext, Database: database, Table: t.Table.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
			bext.metricCh <- MetricUnit{Name: MetricReplDDLTable, Value: 1, LabelPair: map[string]string{"database": database, "table": t.Table.Name.O}}
			bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationDDLTable, Value: 1}
			bext.Logger.Debug("Operation[ddltable], DDL:TableCreate, SchemaContext: `%s`, Database: `%s`, Table: `%s`, Query: %s", schemaContext, database, t.Table.Name.O, string(e.Query))
		case *ast.TruncateTableStmt:
			schemaContext := string(e.Schema)

			database := t.Table.Schema.O
			if len(database) == 0 {
				database = schemaContext
			}

			bext.toMoCh(MysqlOperationDDLTable{SchemaContext: schemaContext, Database: database, Table: t.Table.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
			bext.metricCh <- MetricUnit{Name: MetricReplDDLTable, Value: 1, LabelPair: map[string]string{"database": database, "table": t.Table.Name.O}}
			bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationDDLTable, Value: 1}
			bext.Logger.Debug("Operation[ddltable], DDL:TableTruncate, SchemaContext: %s, Database: %s, Table: %s, Query: %s", schemaContext, database, t.Table.Name.O, string(e.Query))
		case *ast.DropIndexStmt:
			schemaContext := string(e.Schema)

			database := t.Table.Schema.O
			if len(database) == 0 {
				database = schemaContext
			}

			bext.toMoCh(MysqlOperationDDLTable{SchemaContext: schemaContext, Database: database, Table: t.Table.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
			bext.metricCh <- MetricUnit{Name: MetricReplDDLTable, Value: 1, LabelPair: map[string]string{"database": schemaContext, "table": t.Table.Name.O}}
			bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationDDLTable, Value: 1}
			bext.Logger.Debug("Operation[ddltable], DDL:TableIndexDrop, SchemaContext: %s, Database: %s, Table: %s, Query: %s", schemaContext, database, t.Table.Name.O, string(e.Query))
		case *ast.CreateIndexStmt:
			schemaContext := string(e.Schema)

			database := t.Table.Schema.O
			if len(database) == 0 {
				database = schemaContext
			}

			bext.toMoCh(MysqlOperationDDLTable{SchemaContext: schemaContext, Database: database, Table: t.Table.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp})
			bext.metricCh <- MetricUnit{Name: MetricReplDDLTable, Value: 1, LabelPair: map[string]string{"database": schemaContext, "table": t.Table.Name.O}}
			bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationDDLTable, Value: 1}
			bext.Logger.Debug("Operation[ddltable], DDL:TableIndexCreate, SchemaContext: %s, Database: %s, Table: %s, Query: %s", schemaContext, database, t.Table.Name.O, string(e.Query))
		case *ast.CreateDatabaseStmt:
			database := t.Name.O

			bext.toMoCh(MysqlOperationDDLDatabase{Database: database, Query: string(e.Query), Timestamp: eh.Timestamp})
			bext.metricCh <- MetricUnit{Name: MetricReplDDLDatabase, Value: 1, LabelPair: map[string]string{"database": database}}
			bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationDDLDatabase, Value: 1}
			bext.Logger.Debug("Operation[ddldatabase], DDL:DatabaseCreate, Database: %s, Query: %s", database, string(e.Query))
		case *ast.AlterDatabaseStmt:
			database := t.Name.O

			bext.toMoCh(MysqlOperationDDLDatabase{Database: database, Query: string(e.Query), Timestamp: eh.Timestamp})
			bext.metricCh <- MetricUnit{Name: MetricReplDDLDatabase, Value: 1, LabelPair: map[string]string{"database": database}}
			bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationDDLDatabase, Value: 1}
			bext.Logger.Debug("Operation[ddldatabase], DDL:DatabaseAlter, Database: %s, Query: %s", database, string(e.Query))
		case *ast.DropDatabaseStmt:
			database := t.Name.O

			bext.toMoCh(MysqlOperationDDLDatabase{Database: database, Query: string(e.Query), Timestamp: eh.Timestamp})
			bext.metricCh <- MetricUnit{Name: MetricReplDDLDatabase, Value: 1, LabelPair: map[string]string{"database": database}}
			bext.metricCh <- MetricUnit{Name: MetricReplExtractorOperationDDLDatabase, Value: 1}
			bext.Logger.Debug("Operation[ddldatabase], DDL:DatabaseDrop, Database: %s, Query: %s", database, string(e.Query))
		case *ast.BeginStmt:
			bext.toMoCh(MysqlOperationBegin{Timestamp: eh.Timestamp})
			bext.Logger.Debug("Operation[begin]")
		case *ast.CommitStmt:
			bext.Logger.Warning("event query: %s", string(e.Query))
		}
	}
	return nil
}
