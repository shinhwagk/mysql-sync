package main

import (
	"fmt"
	"time"
)

func NewMysqlApply(logLevel int, mysqlClient *MysqlClient, hjdb *HJDB) *MysqlApply {
	return &MysqlApply{
		logger: NewLogger(logLevel, "mysql apply"),

		mysqlClient: mysqlClient,
		hjdb:        hjdb,

		GtidSets:               make(map[string]uint64),
		LastGtidSets:           make(map[string]uint64),
		LastServerUUID:         "",
		LastCommitted:          0,
		LastOperationTimestamp: 0,
		AllowCommit:            false,

		metricDestination: &MetricDestination{0, 0, 0, 0, 0, 0},
	}
}

type MysqlApply struct {
	logger      *Logger
	mysqlClient *MysqlClient
	hjdb        *HJDB

	// state
	GtidSets               map[string]uint64
	LastGtidSets           map[string]uint64
	LastServerUUID         string
	LastCommitted          int64
	LastOperationTimestamp uint32
	AllowCommit            bool

	metricDestination *MetricDestination
}

func (ma *MysqlApply) start(moCh <-chan MysqlOperation) {
	ma.logger.Info("started.")
loop:
	for oper := range moCh {
		switch op := oper.(type) {
		case MysqlOperationDDLDatabase:
			if err := ma.OnDDLDatabase(op); err != nil {
				ma.logger.Error(fmt.Sprintf("OnDDLDatabase -- %s", err))
				break loop
			}
		case MysqlOperationDMLInsert:
			if err := ma.OnDMLInsert(op); err != nil {
				ma.logger.Error("MysqlOperationDMLInsert " + err.Error())
				break loop
			}
		case MysqlOperationDMLDelete:
			ma.logger.Debug("MysqlOperationDMLDelete")
			if err := ma.OnDMLDelete(op); err != nil {
				ma.logger.Error("MysqlOperationDMLDelete " + err.Error())
				break loop
			}
		case MysqlOperationDMLUpdate:
			ma.logger.Debug("MysqlOperationDMLUpdate")
			if err := ma.OnDMLUpdate(op); err != nil {
				ma.logger.Error("MysqlOperationDMLUpdate " + err.Error())
				break loop
			}
		case MysqlOperationDDLTable:
			ma.logger.Debug("MysqlOperationDDLTable")
			if err := ma.OnDDLTable(op); err != nil {
				ma.logger.Error("MysqlOperationDDLTable " + err.Error())
				break loop
			}
		case MysqlOperationXid:
			ma.logger.Debug("MysqlOperationXid")
			if err := ma.OnXID(op); err != nil {
				ma.logger.Error("MysqlOperationXid " + err.Error())
				return
			}
		case MysqlOperationGTID:
			ma.logger.Debug("MysqlOperationGTID")
			if err := ma.OnGTID(op); err != nil {
				ma.logger.Error("MysqlOperationGTID " + err.Error())
				break loop
			}
		case MysqlOperationHeartbeat:
			ma.logger.Debug("MysqlOperationHeartbeat")
			if err := ma.OnHeartbeat(op); err != nil {
				ma.logger.Error("MysqlOperationHeartbeat " + err.Error())
				break loop
			}
		case MysqlOperationBegin:
			ma.logger.Debug("MysqlOperationBegin")
			if err := ma.OnBegin(op); err != nil {
				ma.logger.Error("MysqlOperationBegin " + err.Error())
				break loop
			}
		default:
			fmt.Println("xxxx")
		}
	}
	ma.logger.Info("stopped.")
}

func (ma *MysqlApply) OnDMLInsert(op MysqlOperationDMLInsert) error {
	sql, params := op.GenerateSQL()

	ma.logger.Debug(fmt.Sprintf("OnDMLInsert -- query: '%s' params: '%s'", sql, params))

	if err := ma.mysqlClient.ExecuteDML(sql, params); err != nil {
		return err
	}

	ma.metricDestination.DMLInsertTimes++

	return nil
}

func (ma *MysqlApply) OnDMLDelete(op MysqlOperationDMLDelete) error {
	sql, params := op.GenerateSQL()

	ma.logger.Debug(fmt.Sprintf("OnDMLDelete -- query: '%s' params: '%s'", sql, params))

	if err := ma.mysqlClient.ExecuteDML(sql, params); err != nil {
		return err
	}

	ma.metricDestination.DMLDeleteTimes++
	return nil
}

func (ma *MysqlApply) OnDMLUpdate(op MysqlOperationDMLUpdate) error {
	sql, params := op.GenerateSQL()

	ma.logger.Debug(fmt.Sprintf("OnDMLUpdate -- query: '%s' params: '%s'", sql, params))

	if err := ma.mysqlClient.ExecuteDML(sql, params); err != nil {
		return err
	}

	ma.metricDestination.DMLUpdateTimes++
	return nil
}

func (ma *MysqlApply) OnDDLDatabase(op MysqlOperationDDLDatabase) error {
	ma.logger.Debug(fmt.Sprintf("OnDDLDatabase -- query: '%s'", op.Query))

	ma.LastOperationTimestamp = op.Timestamp

	if err := ma.mysqlClient.ExecuteOnDatabase(op.Query); err != nil {
		return err
	}

	ma.metricDestination.DDLDatabaseTimes++
	return ma.Checkpoint()
}

func (ma *MysqlApply) OnDDLTable(op MysqlOperationDDLTable) error {
	ma.logger.Debug(fmt.Sprintf("OnDDLTable -- schema: %s  query: %s", op.Schema, op.Query))

	ma.LastOperationTimestamp = op.Timestamp

	if err := ma.mysqlClient.ExecuteOnTable(op.Schema, op.Query); err != nil {
		ma.logger.Error(fmt.Sprintf("OnDDLTable -- %s", err))
		return err
	}

	ma.metricDestination.DDLTableTimes++
	return ma.Checkpoint()
}

func (ma *MysqlApply) OnXID(op MysqlOperationXid) error {
	ma.logger.Debug(fmt.Sprintf("OnXID"))

	ma.LastOperationTimestamp = op.Timestamp
	ma.AllowCommit = true
	return nil
}

func (ma *MysqlApply) OnBegin(op MysqlOperationBegin) error {
	ma.logger.Debug(fmt.Sprintf("OnBegin"))

	if err := ma.mysqlClient.Begin(); err != nil {
		return err
	}
	return nil
}

func (ma *MysqlApply) OnGTID(op MysqlOperationGTID) error {
	ma.logger.Debug(fmt.Sprintf("OnGTID -- %s:%d", op.ServerUUID, op.TrxID))

	if ma.LastCommitted != op.LastCommitted {
		ma.MergeCommit()
		ma.LastCommitted = op.LastCommitted
	}
	ma.LastGtidSets[op.ServerUUID] = uint64(op.TrxID)
	ma.LastServerUUID = op.ServerUUID
	return nil
}

func (ma *MysqlApply) OnHeartbeat(op MysqlOperationHeartbeat) error {
	ma.LastOperationTimestamp = op.Timestamp
	err := ma.MergeCommit()

	if ma.metricDestination.Delay != 0 {
		ma.pushMetric()
	}

	return err
}

func (ma *MysqlApply) MergeCommit() error {
	var err error

	if ma.AllowCommit {
		ma.logger.Debug(fmt.Sprintf("MergeCommit -- %T", ma.LastGtidSets))
		ma.AllowCommit = false
		err = ma.mysqlClient.Commit()

		if err != nil {
			return err
		}
		err = ma.Checkpoint()
		if err != nil {
			return err
		}
	}
	return nil

}

func (ma *MysqlApply) Checkpoint() error {
	ma.logger.Info(fmt.Sprintf("Checkpoint -- %s:%d", ma.LastServerUUID, ma.LastGtidSets[ma.LastServerUUID]))

	// if err := ma.hjdb.Update(ma.LastGtidSets); err != nil {
	// 	ma.logger.Error(fmt.Sprintf("Checkpoint -- %s", err))
	// }

	return ma.pushMetric()
}

func (ma *MysqlApply) pushMetric() error {
	ma.logger.Info(fmt.Sprintf("pushMetric"))

	ma.metricDestination.Delay = uint64(time.Now().Unix() - int64(ma.LastOperationTimestamp))
	// ma.metricDirector.push(*ma.metricDestination)
	return nil
}
