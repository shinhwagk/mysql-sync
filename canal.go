package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
)

type Canal struct {
	cfg replication.BinlogSyncerConfig
	ch  chan MysqlOperation

	binlogfile string
}

func (a Canal) Run() {
	syncer := replication.NewBinlogSyncer(a.cfg)
	streamer, _ := syncer.StartSync(mysql.Position{Name: "mysql-bin.000001", Pos: 4})
	a.eventHandler(streamer)
}

func (a Canal) StartDestination(gd DestinationHandler, ch <-chan MysqlOperation) error {
	for oper := range ch {
		switch op := oper.(type) {
		case OperationDDLDatabase:
			gd.OnDDLDatabase(op)
		case MysqlOperationDMLInsert:
			gd.OnDMLInsert(op)
		case MysqlOperationDMLDelete:
			gd.OnDMLDelete(op)
		case MysqlOperationDMLUpdate:
			gd.OnDMLUpdate(op)
		case MysqlOperationDDLTable:
			gd.OnDDLTable(op)
		case MysqlOperationXid:
			gd.OnXID(op)
		case MysqlOperationGTID:
			gd.OnGTID(op)
		case MysqlOperationHeartbeat:
			gd.OnHeartbeat(op)
		case MysqlOperationBegin:
			gd.OnBegin(op)
		default:
			// print("other")
		}
	}
	return nil
}
func (a Canal) AddDestination(d DestinationHandler) {
	go a.StartDestination(d, a.ch)
}

func (a Canal) handleEventWriteRows(e *replication.RowsEvent) error {
	for _, row := range e.Rows {
		mod := MysqlOperationDMLInsert{Database: string(e.Table.Schema), Table: string(e.Table.Table), Columns: []MysqlOperationDMLColumn{}, PrimaryKey: e.Table.PrimaryKey}
		for i := 0; i < int(e.Table.ColumnCount); i++ {
			mod.Columns = append(mod.Columns, MysqlOperationDMLColumn{ColumnName: string(e.Table.ColumnName[i]), ColumnType: e.Table.ColumnType[i], ColumnValue: row[i]})
		}
		a.ch <- mod
	}
	return nil
}

func (a Canal) handleEventDeleteRows(e *replication.RowsEvent) error {
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
		a.ch <- mod
	}
	return nil
}

func (a Canal) handleEventUpdateRows(e *replication.RowsEvent) error {
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
			mod.AfterColumns = append(mod.BeforeColumns, MysqlOperationDMLColumn{ColumnName: string(e.Table.ColumnName[i]), ColumnType: e.Table.ColumnType[i], ColumnValue: after_value[i]})
		}
		a.ch <- mod
	}
	return nil
}

func (a Canal) handleEventGtid(serverID uint32, timestamp uint32, e *replication.GTIDEvent) error {
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
				a.ch <- MysqlOperationGTID{e.LastCommitted, serverID, timestamp, parts[0], xid}
			}
		}
		return nil
	}
}

func (a Canal) eventHandler(streamer *replication.BinlogStreamer) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		ev, _ := streamer.GetEvent(ctx)

		switch e := ev.Event.(type) {
		case *replication.RowsEvent:
			switch ev.Header.EventType {
			case replication.WRITE_ROWS_EVENTv0, replication.WRITE_ROWS_EVENTv1, replication.WRITE_ROWS_EVENTv2:
				a.handleEventWriteRows(e)
			case replication.UPDATE_ROWS_EVENTv0, replication.UPDATE_ROWS_EVENTv1, replication.UPDATE_ROWS_EVENTv2:
				a.handleEventUpdateRows(e)
			case replication.DELETE_ROWS_EVENTv0, replication.DELETE_ROWS_EVENTv1, replication.DELETE_ROWS_EVENTv2:
				a.handleEventDeleteRows(e)
			}
		case *replication.TableMapEvent:

		case *replication.GTIDEvent:
			a.handleEventGtid(ev.Header.ServerID, ev.Header.Timestamp, e)
		case *replication.QueryEvent:
			a.handleQueryEvent(e)
		case *replication.XIDEvent:
			a.ch <- MysqlOperationXid{ev.Header.Timestamp}
		case *replication.RotateEvent:
			switch ev.Header.EventType {
			case replication.ROTATE_EVENT:
				if ev.Header.Timestamp >= 1 {
					a.binlogfile = string(e.NextLogName)
				}
			default:
				fmt.Println("other RotateEvent", e)
			}
		case *replication.GenericEvent:
			switch ev.Header.EventType {
			case replication.HEARTBEAT_EVENT:
				fmt.Println("MysqlOperationHeartbeat")
				a.ch <- MysqlOperationHeartbeat{uint32(time.Now().Unix())}
			}
		}
	}
}
