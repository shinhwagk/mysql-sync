package main

import (
	"context"
	"fmt"
	"time"
)

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
	Clean() error
	// OnPushMetric() error
}

type MysqlDestination struct {
	// ctx context.Context

	ctx    context.Context
	cancel context.CancelFunc
	Ch     <-chan MysqlOperation
	Logger *Logger

	Name string

	Hjdb              *HJDB
	MysqlClient       *MysqlClient
	metricDirector    *MetricDirector
	metricDestination *MetricDestination

	// state
	GtidSets               map[string]uint64
	LastGtidSets           map[string]uint64
	LastServerUUID         string
	LastCommitted          int64
	LastOperationTimestamp uint32
	AllowCommit            bool
}

func NewMysqlDestination(ctx context.Context, cancel context.CancelFunc, logger *Logger, name string, ch chan MysqlOperation, dsn string, hjdb *HJDB, md *MetricDirector) *MysqlDestination {
	return &MysqlDestination{
		ctx:    ctx,
		cancel: cancel,
		Ch:     ch,
		Logger: logger,

		Name: name,

		Hjdb:              hjdb,
		MysqlClient:       NewMysqlClient(ctx, logger, dsn),
		metricDirector:    md,
		metricDestination: &MetricDestination{0, 0, 0, 0, 0, 0},

		GtidSets:               make(map[string]uint64),
		LastGtidSets:           make(map[string]uint64),
		LastServerUUID:         "",
		LastCommitted:          0,
		LastOperationTimestamp: 0,
		AllowCommit:            false,
	}
}

func (md *MysqlDestination) Clean() error {
	return nil
}

func (md *MysqlDestination) OnDMLInsert(op MysqlOperationDMLInsert) error {
	sql, params := op.GenerateSQL()

	md.Logger.Debug(md.Name, fmt.Sprintf("OnDMLInsert -- query: '%s' params: '%s'", sql, params))

	if err := md.MysqlClient.ExecuteDML(sql, params); err != nil {
		return err
	}

	md.metricDestination.DMLInsertTimes++

	return nil
}

func (md *MysqlDestination) OnDMLDelete(op MysqlOperationDMLDelete) error {
	sql, params := op.GenerateSQL()

	md.Logger.Debug(md.Name, fmt.Sprintf("OnDMLDelete -- query: '%s' params: '%s'", sql, params))

	if err := md.MysqlClient.ExecuteDML(sql, params); err != nil {
		return err
	}

	md.metricDestination.DMLDeleteTimes++
	return nil
}

func (md *MysqlDestination) OnDMLUpdate(op MysqlOperationDMLUpdate) error {
	sql, params := op.GenerateSQL()

	md.Logger.Debug(md.Name, fmt.Sprintf("OnDMLUpdate -- query: '%s' params: '%s'", sql, params))

	if err := md.MysqlClient.ExecuteDML(sql, params); err != nil {
		return err
	}

	md.metricDestination.DMLUpdateTimes++
	return nil
}

func (md *MysqlDestination) OnDDLDatabase(op OperationDDLDatabase) error {
	md.LastOperationTimestamp = op.Timestamp
	md.Logger.Debug(md.Name, fmt.Sprintf("OnDDLDatabase -- query: %s", op.Query))

	if err := md.MysqlClient.ExecuteOnDatabase(op.Query); err != nil {
		md.Logger.Error(md.Name, fmt.Sprintf("OnDDLDatabase -- %s", err))
		return err
	}

	md.metricDestination.DDLDatabaseTimes++
	return md.Checkpoint()
}

func (md *MysqlDestination) OnDDLTable(op MysqlOperationDDLTable) error {
	md.LastOperationTimestamp = op.Timestamp
	md.Logger.Debug(md.Name, fmt.Sprintf("OnDDLTable -- schema: %s  query: %s", op.Schema, op.Query))

	if err := md.MysqlClient.ExecuteOnTable(op.Schema, op.Query); err != nil {
		md.Logger.Error(md.Name, fmt.Sprintf("OnDDLTable -- %s", err))
		return err
	}

	md.metricDestination.DDLTableTimes++
	return md.Checkpoint()
}

func (md *MysqlDestination) OnXID(op MysqlOperationXid) error {
	md.LastOperationTimestamp = op.Timestamp
	md.Logger.Debug(md.Name, fmt.Sprintf("OnXID"))
	md.AllowCommit = true
	return nil
}

func (md *MysqlDestination) OnBegin(op MysqlOperationBegin) error {
	md.Logger.Debug(md.Name, fmt.Sprintf("OnBegin"))

	if err := md.MysqlClient.Begin(); err != nil {
		return err
	}
	return nil
}

func (md *MysqlDestination) OnGTID(op MysqlOperationGTID) error {
	md.Logger.Info(md.Name, fmt.Sprintf("OnGTID -- %s:%d", op.ServerUUID, op.TrxID))

	if md.LastCommitted != op.LastCommitted {
		md.MergeCommit()
		md.LastCommitted = op.LastCommitted
	}
	md.LastGtidSets[op.ServerUUID] = uint64(op.TrxID)
	md.LastServerUUID = op.ServerUUID
	return nil
}

func (md *MysqlDestination) OnHeartbeat(op MysqlOperationHeartbeat) error {
	md.LastOperationTimestamp = op.Timestamp
	err := md.MergeCommit()

	if md.metricDestination.Delay != 0 {
		md.pushMetric()
	}

	return err
}

func (md *MysqlDestination) pushMetric() error {
	md.Logger.Info(md.Name, fmt.Sprintf("pushMetric"))

	md.metricDestination.Delay = uint64(time.Now().Unix() - int64(md.LastOperationTimestamp))
	md.metricDirector.push(md.Name, *md.metricDestination)
	return nil
}

func (md *MysqlDestination) Checkpoint() error {
	md.Logger.Info(md.Name, fmt.Sprintf("Checkpoint -- %s:%d", md.LastServerUUID, md.LastGtidSets[md.LastServerUUID]))

	if err := md.Hjdb.Update(md.Name, md.LastGtidSets); err != nil {
		md.Logger.Error(md.Name, fmt.Sprintf("Checkpoint -- %s", err))
	}

	return md.pushMetric()
}

func (md *MysqlDestination) MergeCommit() error {
	var err error

	if md.AllowCommit {
		md.Logger.Debug(md.Name, fmt.Sprintf("MergeCommit -- %T", md.LastGtidSets))
		md.AllowCommit = false
		err = md.MysqlClient.Commit()

		if err != nil {
			return err
		}
		err = md.Checkpoint()
		if err != nil {
			return err
		}
	}
	return nil

}

type KafkaDispatcher struct {
	DestinationHandler
}
