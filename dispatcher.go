package main

import "fmt"

type DestinationHandler interface {
	OnDMLInsert(op MysqlOperationDMLInsert) error
	OnDMLDelete(op MysqlOperationDMLDelete) error
	OnDMLUpdate(op MysqlOperationDMLUpdate) error
	OnDDLDatabase(op OperationDDLDatabase) error
	OnDDLTable(op MysqlOperationDDLTable) error
	OnXID(op MysqlOperationXid) error
	OnGTID(op MysqlOperationGTID) error
	OnHeartbeat(p MysqlOperationHeartbeat) error
	OnBegin(op MysqlOperationBegin) error
}

// type GenericDestination struct{}

// func (gd *GenericDestination) OnDMLInsert(op MysqlOperationDMLInsert) error { return nil }
// func (gd *GenericDestination) OnDMLDelete(op MysqlOperationDMLDelete) error { return nil }
// func (gd *GenericDestination) OnDMLUpdate(op MysqlOperationDMLUpdate) error { return nil }
// func (gd *GenericDestination) OnDDLDatabase(op OperationDDLDatabase) error  { return nil }
// func (gd *GenericDestination) OnDDLTable(op MysqlOperationDDLTable) error   { return nil }
// func (gd *GenericDestination) OnXID(op MysqlOperationXid) error             { return nil }
// func (gd *GenericDestination) OnGTID(op MysqlOperationGTID) error           { return nil }
// func (gd *GenericDestination) OnBegin(op MysqlOperationBegin) error         { return nil }
// func (gd *GenericDestination) OnHeartbeat(p MysqlOperationHeartbeat) error  { return nil }

// MysqlDestination 嵌入 GenericDispatcher
type MysqlDestination struct {
	// *GenericDestination
	ch         <-chan MysqlOperation
	Name       string
	checkpoint string
	Client     *MysqlClient
}

// KafkaDispatcher 嵌入 GenericDispatcher
// type KafkaDispatcher struct {
// 	GenericDispatcher
// }

// func (md DispatcherHandler) OnDMLInsert(op *OperationDMLInsert) error     { return nil }
// func (md DispatcherHandler) OnDMLDelete(op *OperationDMLDelete) error     { return nil }
// func (md DispatcherHandler) OnDMLUpdate(op *OperationDMLUpdate) error     { return nil }
// func (md DispatcherHandler) OnDDLDatabase(op *OperationDDLDatabase) error { return nil }
// func (md DispatcherHandler) OnDDLTable(op *OperationDDLTable) error       { return nil }
// func (md DispatcherHandler) OnXID(op *OperationXid) error                 { return nil }
// func (md DispatcherHandler) OnGTID(op *OperationGtid) error               { return nil }
// func (md DispatcherHandler) OnHeartbeat(p *OperationHeartbeat) error      { return nil }

// type MysqlDispatcher struct {
// 	// DispatcherHandler
// }

func (md MysqlDestination) OnDMLInsert(op MysqlOperationDMLInsert) error {
	fmt.Println("shoudao insert", op.Database, op.Table, op.Columns, op.PrimaryKey)
	sql, params := op.GenerateSQL()
	fmt.Println("genasql", sql, params)
	err := md.Client.ExecuteDML(sql, params)
	fmt.Println("OnDMLInsert", err)
	return nil
}

func (md MysqlDestination) OnDMLDelete(op MysqlOperationDMLDelete) error {
	return nil
}

func (md MysqlDestination) OnDMLUpdate(op MysqlOperationDMLUpdate) error {
	fmt.Println("shoudao update ", op)
	return nil
}

func (md MysqlDestination) OnDDLDatabase(op OperationDDLDatabase) error {
	return nil
}

func (md MysqlDestination) OnDDLTable(op MysqlOperationDDLTable) error {
	fmt.Println(md.Name, "dip", op)
	return nil
}

func (md MysqlDestination) OnXID(op MysqlOperationXid) error {
	fmt.Println(md.Name, "shoudao xid")
	err := md.Client.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (md MysqlDestination) OnBegin(op MysqlOperationBegin) error {
	fmt.Println("shoudao gtid", op)

	err := md.Client.Begin()
	if err != nil {
		return err
	}
	return nil
}

func (md MysqlDestination) OnGTID(op MysqlOperationGTID) error {
	return nil
}

func (md MysqlDestination) OnHeartbeat(p MysqlOperationHeartbeat) error {
	return nil
}

// func (md MysqlDestination) Start(ch <-chan MysqlOperation) error {
// 	for oper := range ch {
// 		switch op := oper.(type) {
// 		case OperationDDLDatabase:
// 			md.OnDDLDatabase(op)
// 		case MysqlOperationDMLInsert:
// 			md.OnDMLInsert(op)
// 		case MysqlOperationDMLDelete:
// 			md.OnDMLDelete(op)
// 		case MysqlOperationDMLUpdate:
// 			md.OnDMLUpdate(op)
// 		case MysqlOperationDDLTable:
// 			md.OnDDLTable(op)
// 		case MysqlOperationXid:
// 			md.OnXID(op)
// 		case MysqlOperationGTID:
// 			md.OnGTID(op)
// 		case MysqlOperationHeartbeat:
// 			md.OnHeartbeat(op)
// 		case MysqlOperationBegin:
// 			md.OnBegin(op)
// 		default:
// 			// print("other")
// 		}
// 	}
// 	return nil
// }

type KafkaDispatcher struct {
	DestinationHandler
}

// func (c *Canal) runSyncBinlog() error {
// 	s, err := c.syncer.StartSync()
// 	if err != nil {
// 		return err
// 	}

// 	for {
// 		ev, err := s.GetEvent(c.ctx)

// 		if err != nil {
// 			return err
// 		}

// 		err = c.handleEvent(ev)
// 		if err != nil {
// 			return err
// 		}
// 	}
// }

// func (c *Canal) handleEvent(ev *replication.BinlogEvent) error {
// 	// savePos := false
// 	// force := false
// 	// pos := c.master.Position()
// 	var err error

// 	// curPos := pos.Pos

// 	// next binlog pos
// 	// pos.Pos = ev.Header.LogPos

// 	// We only save position with RotateEvent and XIDEvent.
// 	// For RowsEvent, we can't save the position until meeting XIDEvent
// 	// which tells the whole transaction is over.
// 	// TODO: If we meet any DDL query, we must save too.
// 	switch e := ev.Event.(type) {
// 	case *replication.RotateEvent:
// 		// pos.Name = string(e.NextLogName)
// 		// pos.Pos = uint32(e.Position)
// 		// c.cfg.Logger.Infof("rotate binlog to %s", pos)
// 		// savePos = true
// 		// force = true
// 		if err = c.eventHandler.OnRotate(ev.Header, e); err != nil {
// 			return errors.Trace(err)
// 		}
// 	case *replication.RowsEvent:
// 		// we only focus row based event
// 		err = c.handleRowsEvent(ev)
// 		if err != nil {
// 			// c.cfg.Logger.Errorf("handle rows event at (%s, %d) error %v", pos.Name, curPos, err)
// 			return errors.Trace(err)
// 		}
// 		return nil
// 	case *replication.TransactionPayloadEvent:
// 		// handle subevent row by row
// 		ev := ev.Event.(*replication.TransactionPayloadEvent)
// 		for _, subEvent := range ev.Events {
// 			err = c.handleEvent(subEvent)
// 			if err != nil {
// 				// c.cfg.Logger.Errorf("handle transaction payload subevent at (%s, %d) error %v", pos.Name, curPos, err)
// 				return errors.Trace(err)
// 			}
// 		}
// 		return nil
// 	case *replication.XIDEvent:
// 		// savePos = true

// 		// try to save the position later
// 		if err := c.eventHandler.OnXID(ev.Header, pos); err != nil {
// 			return errors.Trace(err)
// 		}
// 		// if e.GSet != nil {
// 		// 	// c.master.UpdateGTIDSet(e.GSet)
// 		// }
// 	case *replication.MariadbGTIDEvent:
// 		if err := c.eventHandler.OnGTID(ev.Header, e); err != nil {
// 			return errors.Trace(err)
// 		}
// 	case *replication.GTIDEvent:
// 		if err := c.eventHandler.OnGTID(ev.Header, e); err != nil {
// 			return errors.Trace(err)
// 		}
// 	// case *replication.RowsQueryEvent:
// 	// 	if err := c.eventHandler.OnRowsQueryEvent(e); err != nil {
// 	// 		return errors.Trace(err)
// 	// 	}
// 	case *replication.QueryEvent:
// 		stmts, _, err := c.parser.Parse(string(e.Query), "", "")
// 		if err != nil {
// 			c.cfg.Logger.Errorf("parse query(%s) err %v, will skip this event", e.Query, err)
// 			return nil
// 		}
// 		for _, stmt := range stmts {
// 			nodes := parseStmt(stmt)
// 			for _, node := range nodes {
// 				if node.db == "" {
// 					node.db = string(e.Schema)
// 				}
// 				if err = c.updateTable(ev.Header, node.db, node.table); err != nil {
// 					return errors.Trace(err)
// 				}
// 			}
// 			if len(nodes) > 0 {

// 				if err = c.eventHandler.OnNonDML(ev.Header, pos, e); err != nil {
// 					return errors.Trace(err)
// 				}
// 			}
// 		}

// 	default:
// 		return nil
// 	}

// 	return nil
// }
