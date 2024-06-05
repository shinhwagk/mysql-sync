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

	// _ "github.com/pingcap/tidb/pkg/parser/test_driver"
	_ "github.com/pingcap/tidb/pkg/types/parser_driver"
)

type Canal struct {
	Name string

	cfg  replication.BinlogSyncerConfig
	gtid string

	ch chan MysqlOperation

	binlogfile string

	ctx    context.Context
	cancel context.CancelFunc

	Logger *Logger
}

func NewCanal(ctx context.Context, cancel context.CancelFunc, cfg replication.BinlogSyncerConfig, gtid string, ch chan MysqlOperation, logger *Logger) *Canal {
	return &Canal{
		Name:   "Canal",
		cfg:    cfg,
		gtid:   gtid,
		ch:     ch,
		ctx:    ctx,
		cancel: cancel,
		Logger: logger,
	}
}

func (c Canal) handleQueryEvent(e *replication.QueryEvent, eh *replication.EventHeader) error {
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
			c.ch <- MysqlOperationDDLTable{Schema: schema, Table: t.Table.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp}
		case *ast.DropTableStmt:
			schema := string(e.Schema)
			for _, tab := range t.Tables {
				if len(schema) == 0 {
					schema = tab.Schema.O
				}
				c.ch <- MysqlOperationDDLTable{Schema: schema, Table: tab.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp}
			}
		case *ast.CreateTableStmt:
			schema := string(e.Schema)
			if len(schema) == 0 {
				schema = t.Table.Schema.O
			}
			c.ch <- MysqlOperationDDLTable{Schema: schema, Table: t.Table.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp}
		case *ast.TruncateTableStmt:
			schema := string(e.Schema)
			if len(schema) == 0 {
				schema = t.Table.Schema.O
			}
			c.ch <- MysqlOperationDDLTable{Schema: schema, Table: t.Table.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp}
		case *ast.CreateIndexStmt:
			schema := string(e.Schema)
			if len(schema) == 0 {
				schema = t.Table.Schema.O
			}
			c.ch <- MysqlOperationDDLTable{Schema: schema, Table: t.Table.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp}
		case *ast.DropIndexStmt:
		case *ast.CreateDatabaseStmt:
			c.ch <- OperationDDLDatabase{Schema: t.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp}
		case *ast.AlterDatabaseStmt:
			c.ch <- OperationDDLDatabase{Schema: t.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp}
		case *ast.DropDatabaseStmt:
			c.ch <- OperationDDLDatabase{Schema: t.Name.O, Query: string(e.Query), Timestamp: eh.Timestamp}
		case *ast.BeginStmt:
			c.ch <- MysqlOperationBegin{}
		case *ast.CommitStmt:
			// warning
		}
	}
	return nil
}

func (c Canal) Run() {
	// streamer, _ := syncer.StartSync(mysql.Position{Name: "mysql-bin.000001", Pos: 4})
	if gtidSet, err := mysql.ParseMysqlGTIDSet(c.gtid); err != nil {

	} else {
		defer c.cancel()

		syncer := replication.NewBinlogSyncer(c.cfg)
		streamer, _ := syncer.StartSyncGTID(gtidSet)

		c.Logger.Info("canal", "Running")
		c.eventHandler(streamer)
	}
}

func (c Canal) AddDestination(d DestinationHandler) {
	go func() {
		c.startDestination(d, c.ch)
		c.Logger.Info("canal", "close destination")
		c.cancel()
	}()
}

func (c Canal) startDestination(gd DestinationHandler, ch <-chan MysqlOperation) {
	for oper := range ch {
		switch op := oper.(type) {
		case OperationDDLDatabase:
			c.Logger.Debug("canal", "OperationDDLDatabase")
			if err := gd.OnDDLDatabase(op); err != nil {
				c.Logger.Error(c.Name, fmt.Sprintf("OperationDDLDatabase %s", err))
				return
			}
		case MysqlOperationDMLInsert:
			c.Logger.Debug("canal", "MysqlOperationDMLInsert")
			if err := gd.OnDMLInsert(op); err != nil {
				c.Logger.Error("canal", "MysqlOperationDMLInsert "+err.Error())
				return
			}
		case MysqlOperationDMLDelete:
			c.Logger.Debug("canal", "MysqlOperationDMLDelete")
			if err := gd.OnDMLDelete(op); err != nil {
				c.Logger.Error("canal", "MysqlOperationDMLDelete "+err.Error())
				return
			}
		case MysqlOperationDMLUpdate:
			c.Logger.Debug("canal", "MysqlOperationDMLUpdate")
			if err := gd.OnDMLUpdate(op); err != nil {
				c.Logger.Error("canal", "MysqlOperationDMLUpdate "+err.Error())
				return
			}
		case MysqlOperationDDLTable:
			c.Logger.Debug("canal", "MysqlOperationDDLTable")
			if err := gd.OnDDLTable(op); err != nil {
				c.Logger.Error("canal", "MysqlOperationDDLTable "+err.Error())
				return
			}
		case MysqlOperationXid:
			c.Logger.Debug("canal", "MysqlOperationXid")
			if err := gd.OnXID(op); err != nil {
				c.Logger.Error("canal", "MysqlOperationXid "+err.Error())
				return
			}
		case MysqlOperationGTID:
			c.Logger.Debug("canal", "MysqlOperationGTID")
			if err := gd.OnGTID(op); err != nil {
				c.Logger.Error("canal", "MysqlOperationGTID "+err.Error())
				return
			}
		case MysqlOperationHeartbeat:
			c.Logger.Debug("canal", "MysqlOperationHeartbeat")
			if err := gd.OnHeartbeat(op); err != nil {
				c.Logger.Error("canal", "MysqlOperationHeartbeat "+err.Error())
				return
			}
		case MysqlOperationBegin:
			c.Logger.Debug("canal", "MysqlOperationBegin")
			if err := gd.OnBegin(op); err != nil {
				c.Logger.Error("canal", "MysqlOperationBegin "+err.Error())
				return
			}
		default:
		}
	}
}

func (c Canal) handleEventWriteRows(e *replication.RowsEvent) error {
	for _, row := range e.Rows {
		mod := MysqlOperationDMLInsert{Database: string(e.Table.Schema), Table: string(e.Table.Table), Columns: []MysqlOperationDMLColumn{}, PrimaryKey: e.Table.PrimaryKey}
		for i := 0; i < int(e.Table.ColumnCount); i++ {
			mod.Columns = append(mod.Columns, MysqlOperationDMLColumn{ColumnName: string(e.Table.ColumnName[i]), ColumnType: e.Table.ColumnType[i], ColumnValue: row[i]})
		}
		c.ch <- mod
	}
	return nil
}

func (c Canal) handleEventDeleteRows(e *replication.RowsEvent) error {
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
		c.ch <- mod
	}
	return nil
}

func (c Canal) handleEventUpdateRows(e *replication.RowsEvent) error {
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
		c.ch <- mod
	}
	return nil
}

func (c Canal) handleEventGtid(e *replication.GTIDEvent, eh *replication.EventHeader) error {
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
				c.ch <- MysqlOperationGTID{e.LastCommitted, eh.ServerID, eh.Timestamp, parts[0], xid}
			}
		}
		return nil
	}
}

func (c Canal) eventHandler(streamer *replication.BinlogStreamer) {
	for {
		ev, err := streamer.GetEvent(c.ctx)

		if err != nil {
			if err == context.DeadlineExceeded {
				fmt.Println("Event fetch timed out:", err)
				continue
			} else if err == context.Canceled {
				fmt.Println("Event handling canceled:", err)
				return
			}
			fmt.Println("Error getting event:", err)
			continue
		}

		switch e := ev.Event.(type) {
		case *replication.RowsEvent:
			switch ev.Header.EventType {
			case replication.WRITE_ROWS_EVENTv0, replication.WRITE_ROWS_EVENTv1, replication.WRITE_ROWS_EVENTv2:
				if err := c.handleEventWriteRows(e); err != nil {
					fmt.Println(err)
					break
				}
			case replication.UPDATE_ROWS_EVENTv0, replication.UPDATE_ROWS_EVENTv1, replication.UPDATE_ROWS_EVENTv2:
				c.handleEventUpdateRows(e)
			case replication.DELETE_ROWS_EVENTv0, replication.DELETE_ROWS_EVENTv1, replication.DELETE_ROWS_EVENTv2:
				c.handleEventDeleteRows(e)
			}
		case *replication.TableMapEvent:
		case *replication.GTIDEvent:
			c.handleEventGtid(e, ev.Header)
		case *replication.QueryEvent:
			c.handleQueryEvent(e, ev.Header)
		case *replication.XIDEvent:
			c.ch <- MysqlOperationXid{ev.Header.Timestamp}
		case *replication.RotateEvent:
			switch ev.Header.EventType {
			case replication.ROTATE_EVENT:
				if ev.Header.Timestamp >= 1 {
					c.binlogfile = string(e.NextLogName)
				}
			default:
				fmt.Println("other RotateEvent", e)
			}
		case *replication.GenericEvent:
			switch ev.Header.EventType {
			case replication.HEARTBEAT_EVENT:
				fmt.Println("MysqlOperationHeartbeat")
				c.ch <- MysqlOperationHeartbeat{uint32(time.Now().Unix())}
			}
		}
	}
}
