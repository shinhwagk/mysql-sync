package canal

import (
	"context"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mysql-org/go-mysql/client"
	"github.com/go-mysql-org/go-mysql/dump"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
	"github.com/go-mysql-org/go-mysql/schema"
	"github.com/pingcap/errors"
	"github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"
)

type Canal struct {
	cfg        *Config
	gset       mysql.GTIDSet
	parser     *parser.Parser
	dumper     *dump.Dumper
	dumped     bool
	dumpDoneCh chan struct{}
	syncer     *replication.BinlogSyncer

	eventHandler EventHandler

	connLock sync.Mutex
	conn     *client.Conn

	tableLock          sync.RWMutex
	tables             map[string]*schema.Table
	errorTablesGetTime map[string]time.Time

	tableMatchCache   map[string]bool
	includeTableRegex []*regexp.Regexp
	excludeTableRegex []*regexp.Regexp

	delay *uint32

	ctx    context.Context
	cancel context.CancelFunc
}

func (c *Canal) startSyncer() (*replication.BinlogStreamer, error) {

	s, err := c.syncer.StartSyncGTID(c.gset)
	if err != nil {
		return nil, errors.Errorf("start sync replication at GTID set %v error %v", c.gset, err)
	}

	return s, nil

}

func (c *Canal) runSyncBinlog() error {
	s, err := c.startSyncer()
	if err != nil {
		return err
	}

	for {
		ev, err := s.GetEvent(c.ctx)
		if err != nil {
			return errors.Trace(err)
		}

		// Update the delay between the Canal and the Master before the handler hooks are called
		c.updateReplicationDelay(ev)

		// If log pos equals zero then the received event is a fake rotate event and
		// contains only a name of the next binlog file
		// See https://github.com/mysql/mysql-server/blob/8e797a5d6eb3a87f16498edcb7261a75897babae/sql/rpl_binlog_sender.h#L235
		// and https://github.com/mysql/mysql-server/blob/8cc757da3d87bf4a1f07dcfb2d3c96fed3806870/sql/rpl_binlog_sender.cc#L899
		// if ev.Header.LogPos == 0 {
		// 	switch e := ev.Event.(type) {
		// 	case *replication.RotateEvent:
		// 		continue
		// 	default:
		// 		continue
		// 	}
		// }

		err = c.handleEvent(ev)
		if err != nil {
			return err
		}
	}
}

func (c *Canal) handleEvent(ev *replication.BinlogEvent) error {
	// savePos := false
	// force := false
	// pos := c.master.Position()
	var err error

	// curPos := pos.Pos

	// next binlog pos
	// pos.Pos = ev.Header.LogPos

	// We only save position with RotateEvent and XIDEvent.
	// For RowsEvent, we can't save the position until meeting XIDEvent
	// which tells the whole transaction is over.
	// TODO: If we meet any DDL query, we must save too.
	switch e := ev.Event.(type) {
	case *replication.RotateEvent:
		// pos.Name = string(e.NextLogName)
		// pos.Pos = uint32(e.Position)
		// c.cfg.Logger.Infof("rotate binlog to %s", pos)
		// savePos = true
		// force = true
		if err = c.eventHandler.OnRotate(ev.Header, e); err != nil {
			return errors.Trace(err)
		}
	case *replication.RowsEvent:
		// we only focus row based event
		err = c.handleRowsEvent(ev)
		if err != nil {
			// c.cfg.Logger.Errorf("handle rows event at (%s, %d) error %v", pos.Name, curPos, err)
			return errors.Trace(err)
		}
		return nil
	case *replication.TransactionPayloadEvent:
		// handle subevent row by row
		ev := ev.Event.(*replication.TransactionPayloadEvent)
		for _, subEvent := range ev.Events {
			err = c.handleEvent(subEvent)
			if err != nil {
				// c.cfg.Logger.Errorf("handle transaction payload subevent at (%s, %d) error %v", pos.Name, curPos, err)
				return errors.Trace(err)
			}
		}
		return nil
	case *replication.XIDEvent:
		// savePos = true

		// try to save the position later
		if err := c.eventHandler.OnXID(ev.Header, pos); err != nil {
			return errors.Trace(err)
		}
		// if e.GSet != nil {
		// 	// c.master.UpdateGTIDSet(e.GSet)
		// }
	case *replication.MariadbGTIDEvent:
		if err := c.eventHandler.OnGTID(ev.Header, e); err != nil {
			return errors.Trace(err)
		}
	case *replication.GTIDEvent:
		if err := c.eventHandler.OnGTID(ev.Header, e); err != nil {
			return errors.Trace(err)
		}
	// case *replication.RowsQueryEvent:
	// 	if err := c.eventHandler.OnRowsQueryEvent(e); err != nil {
	// 		return errors.Trace(err)
	// 	}
	case *replication.QueryEvent:
		stmts, _, err := c.parser.Parse(string(e.Query), "", "")
		if err != nil {
			c.cfg.Logger.Errorf("parse query(%s) err %v, will skip this event", e.Query, err)
			return nil
		}
		for _, stmt := range stmts {
			nodes := parseStmt(stmt)
			for _, node := range nodes {
				if node.db == "" {
					node.db = string(e.Schema)
				}
				if err = c.updateTable(ev.Header, node.db, node.table); err != nil {
					return errors.Trace(err)
				}
			}
			if len(nodes) > 0 {

				if err = c.eventHandler.OnNonDML(ev.Header, pos, e); err != nil {
					return errors.Trace(err)
				}
			}
		}

	default:
		return nil
	}

	return nil
}

type node struct {
	db    string
	table string
}

func parseStmt(stmt ast.StmtNode) (ns []*node) {
	switch t := stmt.(type) {
	case *ast.RenameTableStmt:
		ns = make([]*node, len(t.TableToTables))
		for i, tableInfo := range t.TableToTables {
			ns[i] = &node{
				db:    tableInfo.OldTable.Schema.String(),
				table: tableInfo.OldTable.Name.String(),
			}
		}
	case *ast.AlterTableStmt:
		n := &node{
			db:    t.Table.Schema.String(),
			table: t.Table.Name.String(),
		}
		ns = []*node{n}
	case *ast.DropTableStmt:
		ns = make([]*node, len(t.Tables))
		for i, table := range t.Tables {
			ns[i] = &node{
				db:    table.Schema.String(),
				table: table.Name.String(),
			}
		}
	case *ast.CreateTableStmt:
		n := &node{
			db:    t.Table.Schema.String(),
			table: t.Table.Name.String(),
		}
		ns = []*node{n}
	case *ast.TruncateTableStmt:
		n := &node{
			db:    t.Table.Schema.String(),
			table: t.Table.Name.String(),
		}
		ns = []*node{n}
	case *ast.CreateIndexStmt:
		n := &node{
			db:    t.Table.Schema.String(),
			table: t.Table.Name.String(),
		}
		ns = []*node{n}
	case *ast.DropIndexStmt:
		n := &node{
			db:    t.Table.Schema.String(),
			table: t.Table.Name.String(),
		}
		ns = []*node{n}
	case *ast.CreateDatabaseStmt:
		n := &node{
			db:    t.Name.String(),
			table: "",
		}
		ns = []*node{n}
	}

	return ns
}

func (c *Canal) updateTable(header *replication.EventHeader, db, table string) (err error) {
	c.ClearTableCache([]byte(db), []byte(table))
	c.cfg.Logger.Infof("table structure changed, clear table cache: %s.%s\n", db, table)
	if err = c.eventHandler.OnTableChanged(header, db, table); err != nil && errors.Cause(err) != schema.ErrTableNotExist {
		return errors.Trace(err)
	}
	return
}
func (c *Canal) updateReplicationDelay(ev *replication.BinlogEvent) {
	var newDelay uint32
	now := uint32(time.Now().Unix())
	if now >= ev.Header.Timestamp {
		newDelay = now - ev.Header.Timestamp
	}
	atomic.StoreUint32(c.delay, newDelay)
}

func (c *Canal) handleRowsEvent(e *replication.BinlogEvent) error {
	ev := e.Event.(*replication.RowsEvent)

	// Caveat: table may be altered at runtime.
	schemaName := string(ev.Table.Schema)
	tableName := string(ev.Table.Table)

	t, err := c.GetTable(schemaName, tableName)
	if err != nil {
		e := errors.Cause(err)
		// ignore errors below
		if e == ErrExcludedTable || e == schema.ErrTableNotExist || e == schema.ErrMissingTableMeta {
			err = nil
		}

		return err
	}
	var action string
	switch e.Header.EventType {
	case replication.WRITE_ROWS_EVENTv1, replication.WRITE_ROWS_EVENTv2, replication.MARIADB_WRITE_ROWS_COMPRESSED_EVENT_V1:
		action = InsertAction
	case replication.DELETE_ROWS_EVENTv1, replication.DELETE_ROWS_EVENTv2, replication.MARIADB_DELETE_ROWS_COMPRESSED_EVENT_V1:
		action = DeleteAction
	case replication.UPDATE_ROWS_EVENTv1, replication.UPDATE_ROWS_EVENTv2, replication.MARIADB_UPDATE_ROWS_COMPRESSED_EVENT_V1:
		action = UpdateAction
	default:
		return errors.Errorf("%s not supported now", e.Header.EventType)
	}

	events := newRowsEvent(t, action, ev.Rows, e.Header)
	return c.eventHandler.OnRow(events)
}
