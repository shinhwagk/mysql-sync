package main

import (
	"context"
	"fmt"
	"strings"
)

const (
	StateNULL  = "null"
	StateDML   = "dml"
	StateDDL   = "ddl"
	StateDCL   = "dcl"
	StateBEGIN = "begin"
	StateXID   = "xid"
	StateGTID  = "gtid"
)

func NewMysqlApplier(logLevel int, ckpt *Checkpoint, destConf DestinationConfig, metricCh chan<- MetricUnit) (*MysqlApplier, error) {
	logger := NewLogger(logLevel, "mysql-applier")

	replicateFilter := NewReplicateFilter(destConf.Sync.Replicate)

	if mysqlClient, err := NewMysqlClient(logLevel, destConf.Mysql); err != nil {
		logger.Error("create mysql client: %s", err)
		mysqlClient.Close()
		return nil, err
	} else {
		return &MysqlApplier{
			Logger:                  logger,
			ckpt:                    ckpt,
			mysqlClient:             mysqlClient,
			replicate:               replicateFilter,
			DmlModeUpdate:           destConf.Sync.DmlMode.Update,
			DmlModeInsert:           destConf.Sync.DmlMode.Insert,
			MergeCommitMaxCount:     destConf.Sync.MergeCommit.MaxCount,
			MergeCommitMaxDelay:     destConf.Sync.MergeCommit.MaxDelay,
			LastCommitted:           0,
			LastCheckpointTimestamp: 0,
			LastAppliedTimestamp:    0,
			PendingCommitCount:      0,
			GtidCount:               0,
			metricCh:                metricCh,
			GtidSkip:                false,
			State:                   StateNULL,
		}, nil
	}
}

type MysqlApplier struct {
	Logger                  *Logger
	mysqlClient             *MysqlClient
	replicate               *Replicate
	ckpt                    *Checkpoint
	DmlModeUpdate           string
	DmlModeInsert           string
	MergeCommitMaxCount     uint
	MergeCommitMaxDelay     uint32
	LastCommitted           int64
	LastCheckpointTimestamp uint32
	LastAppliedTimestamp    uint32
	PendingCommitCount      uint
	GtidCount               uint
	metricCh                chan<- MetricUnit
	GtidSkip                bool
	State                   string
}

func (ma *MysqlApplier) ReplicateNotExecute(schemaContext string, tableName string) bool {
	return !ma.replicate.filter(schemaContext, tableName)
}

func (ma *MysqlApplier) Start(ctx context.Context, moCh <-chan MysqlOperation) {
	ma.Logger.Info("started.")
	defer ma.Logger.Info("stopped.")

	for {
		select {
		// case <-time.After(time.Millisecond * 100):
		case <-ctx.Done():
			ma.Logger.Info("ctx done signal received.")
			if err := ma.mysqlClient.Rollback(); err != nil {
				ma.mysqlClient.Logger.Error("mysql connect rollback: %s.", err)
			} else {
				ma.mysqlClient.Logger.Info("mysql connect rollback complate.")
			}
			if err := ma.mysqlClient.Close(); err != nil {
				ma.mysqlClient.Logger.Error("mysql connect close: %s.", err)
			} else {
				ma.mysqlClient.Logger.Info("mysql connect close complate.")
			}
			return
		case oper := <-moCh:
			ma.metricCh <- MetricUnit{Name: MetricDestApplierOperationsCache, Value: uint(len(moCh))}

			if ma.GtidSkip {
				if _, ok := oper.(MysqlOperationGTID); !ok {
					continue
				}
				ma.State = StateNULL
			}

			switch op := oper.(type) {
			case MysqlOperationBinLogPos:
				ma.Logger.Debug("Operation[binlogpos] -- timestamp: %d, server id: %d, file: %s, pos: %d, event: %s, ", op.GetTimestamp(), op.ServerID, op.File, op.Pos, op.Event)

				ma.ckpt.SetBinlogPos(op.File, op.Pos)

				continue
			case MysqlOperationHeartbeat:
				ma.LastCheckpointTimestamp = ma.LastAppliedTimestamp

				ma.Logger.Trace("Operation[heartbeat]")

				if ma.State == StateDDL || ma.State == StateDCL {
					ma.Checkpoint()
				} else if ma.State == StateXID {
					if err := ma.MergeCommit(); err != nil {
						return
					}
				} else if ma.State == StateNULL {
					ma.metricCh <- MetricUnit{Name: MetricDestCheckpointTimestamp, Value: uint(ma.LastCheckpointTimestamp)}
				} else {
					ma.Logger.Error("Execute[heartbeat] -- last state is '%s'", ma.State)
					return
				}

				ma.State = StateNULL
			case MysqlOperationGTID:
				ma.LastCheckpointTimestamp = ma.LastAppliedTimestamp

				ma.Logger.Debug("Operation[gtid] -- gtid: %s:%d, lastcommitted: %d", op.ServerUUID, op.TrxID, op.LastCommitted)

				if ma.State == StateDDL || ma.State == StateDCL {
					ma.Checkpoint()
				} else if ma.State == StateXID {
					if ma.EvaluateForceMergeCommit() {
						if err := ma.MergeCommit(); err != nil {
							return
						}
					}
				} else if ma.State == StateNULL {
				} else {
					ma.Logger.Error("Execute[gtid] -- last state is '%s'", ma.State)
					return
				}

				ma.Logger.Debug("Execute[gtid]")
				if err := ma.OnGTID(op); err != nil {
					return
				}

				ma.State = StateGTID
			case MysqlOperationDDLDatabase:
				ma.Logger.Debug("Operation[ddldatabase] -- DDL:Database, Database: %s, Query: %s", op.Database, op.Query)

				if !(ma.State == StateGTID) {
					ma.Logger.Error("Execute[ddldatabase] -- last state is '%s'", ma.State)
					return
				}

				if err := ma.MergeCommit(); err != nil {
					return
				}

				if ma.ReplicateNotExecute(op.Database, "") {
					ma.Logger.Debug("Execute[ddldatabase] -- replicate filter: skip")

					ma.metricCh <- MetricUnit{Name: MetricDestDDLDatabaseSkip, Value: 1, LabelPair: map[string]string{"database": op.Database}}
					ma.metricCh <- MetricUnit{Name: MetricDestApplierOperationDDLDatabaseSkip, Value: 1}
				} else {
					ma.Logger.Debug("Execute[ddldatabase]")

					if err := ma.OnDDLDatabase(op); err != nil {
						return
					}
					ma.metricCh <- MetricUnit{Name: MetricDestDDLDatabase, Value: 1, LabelPair: map[string]string{"database": op.Database}}
					ma.metricCh <- MetricUnit{Name: MetricDestApplierOperationDDLDatabase, Value: 1}
				}

				ma.State = StateDDL
			case MysqlOperationDDLTable:
				ma.Logger.Debug("Operation[ddltable] -- SchemaContext: %s, Database: %s, Table: %s, Query: %s", op.SchemaContext, op.Database, op.Table, op.Query)

				if !(ma.State == StateGTID || ma.State == StateDDL) {
					ma.Logger.Error("Execute[ddltable] -- last state is '%s'", ma.State)
					return
				}

				if err := ma.MergeCommit(); err != nil {
					return
				}

				if ma.ReplicateNotExecute(op.SchemaContext, "") || ma.ReplicateNotExecute(op.Database, op.Table) {
					ma.Logger.Debug("Execute[ddltable] -- replicate filter: skip")

					ma.metricCh <- MetricUnit{Name: MetricDestDDLTableSkip, Value: 1, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
					ma.metricCh <- MetricUnit{Name: MetricDestApplierOperationDDLTableSkip, Value: 1}
				} else {
					ma.Logger.Debug("Execute[ddltable]")

					if err := ma.OnDDLTable(op); err != nil {
						return
					}
					ma.metricCh <- MetricUnit{Name: MetricDestDDLTable, Value: 1, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
					ma.metricCh <- MetricUnit{Name: MetricDestApplierOperationDDLTable, Value: 1}
				}

				ma.State = StateDDL
			case MysqlOperationDCLUser:
				ma.Logger.Debug("Operation[dcluser] -- SchemaContext: %s, Query: %s", op.SchemaContext, op.Query)

				if !(ma.State == StateGTID || ma.State == StateDCL) {
					ma.Logger.Error("Execute[dcluser] -- last state is '%s'", ma.State)
					return
				}

				if err := ma.MergeCommit(); err != nil {
					return
				}

				if ma.ReplicateNotExecute(op.SchemaContext, "") {
					ma.Logger.Debug("Execute[dcluser] -- replicate filter: skip")

					// ma.metricCh <- MetricUnit{Name: MetricDestDDLTableSkip, Value: 1, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
					// ma.metricCh <- MetricUnit{Name: MetricDestApplierOperationDDLTableSkip, Value: 1}
				} else {
					ma.Logger.Debug("Execute[dcluser]")

					if err := ma.OnDCLUser(op); err != nil {
						return
					}
					// ma.metricCh <- MetricUnit{Name: MetricDestDDLTable, Value: 1, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
					// ma.metricCh <- MetricUnit{Name: MetricDestApplierOperationDDLTable, Value: 1}
				}

				ma.State = StateDCL
			case MysqlOperationBegin:
				ma.Logger.Debug("Operation[begin]")

				if !(ma.State == StateGTID) {
					ma.Logger.Error("Execute[begin] -- last state is '%s'", ma.State)
					return
				}

				if ma.PendingCommitCount == 0 {
					if err := ma.OnBegin(op); err != nil {
						return
					}
					ma.Logger.Debug("Execute[begin]")
				} else {
					ma.Logger.Debug("Execute[begin] -- skipped due to merge commits")
				}

				ma.State = StateBEGIN
			case MysqlOperationDMLInsert:
				ma.Logger.Debug("Operation[dmlinsert] -- database: %s, table: %s, mode: %s", op.Database, op.Table, ma.DmlModeInsert)

				if !(ma.State == StateBEGIN || ma.State == StateDML) {
					ma.Logger.Error("Execute[dmlinsert] -- last state is '%s'", ma.State)
					return
				}

				if ma.ReplicateNotExecute(op.Database, op.Table) {
					ma.Logger.Debug("Execute[dmlinsert] -- replicate filter: skip")

					ma.metricCh <- MetricUnit{Name: MetricDestDMLInsertSkip, Value: 1, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
					ma.metricCh <- MetricUnit{Name: MetricDestDMLUpdateSkip, Value: 0, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
					ma.metricCh <- MetricUnit{Name: MetricDestDMLDeleteSkip, Value: 0, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
					ma.metricCh <- MetricUnit{Name: MetricDestApplierOperationDMLInsertSkip, Value: 1}
				} else {
					if gobRepairColumns, err := GobUint8NilRepair(op.Columns); err != nil {
						ma.Logger.Error("Execute[dmlinsert] -- gob repair []uint8(nil): ", err)
						return
					} else {
						op.Columns = gobRepairColumns
					}
					if err := ma.OnDMLInsert(op); err != nil {
						return
					}
					ma.metricCh <- MetricUnit{Name: MetricDestDMLInsert, Value: 1, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
					ma.metricCh <- MetricUnit{Name: MetricDestDMLDelete, Value: 0, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
					ma.metricCh <- MetricUnit{Name: MetricDestDMLUpdate, Value: 0, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
					ma.metricCh <- MetricUnit{Name: MetricDestApplierOperationDMLInsert, Value: 1}
				}

				ma.State = StateDML
			case MysqlOperationDMLDelete:
				ma.Logger.Debug("Operation[dmldelete] -- database: %s, table: %s", op.Database, op.Table)

				if !(ma.State == StateBEGIN || ma.State == StateDML) {
					ma.Logger.Error("Execute[dmldelete] -- last state is '%s'", ma.State)
					return
				}

				if ma.ReplicateNotExecute(op.Database, op.Table) {
					ma.Logger.Debug("Execute[dmldelete] -- replicate filter: skip")

					ma.metricCh <- MetricUnit{Name: MetricDestDMLDeleteSkip, Value: 1, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
					ma.metricCh <- MetricUnit{Name: MetricDestDMLUpdateSkip, Value: 0, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
					ma.metricCh <- MetricUnit{Name: MetricDestDMLInsertSkip, Value: 0, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}

					ma.metricCh <- MetricUnit{Name: MetricDestApplierOperationDMLDeleteSkip, Value: 1}
				} else {
					if gobRepairColumns, err := GobUint8NilRepair(op.Columns); err != nil {
						ma.Logger.Error("Execute[dmldelete] -- gob repair []uint8(nil): ", err)
						return
					} else {
						op.Columns = gobRepairColumns
					}
					if err := ma.OnDMLDelete(op); err != nil {
						return
					}
					ma.metricCh <- MetricUnit{Name: MetricDestDMLInsert, Value: 0, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
					ma.metricCh <- MetricUnit{Name: MetricDestDMLDelete, Value: 1, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
					ma.metricCh <- MetricUnit{Name: MetricDestDMLUpdate, Value: 0, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
					ma.metricCh <- MetricUnit{Name: MetricDestApplierOperationDMLDelete, Value: 1}
				}

				ma.State = StateDML
			case MysqlOperationDMLUpdate:
				ma.Logger.Debug("Operation[dmlupdate] -- database: %s, table: %s, mode: %s", op.Database, op.Table, ma.DmlModeUpdate)

				if !(ma.State == StateBEGIN || ma.State == StateDML) {
					ma.Logger.Error("Execute[dmlupdate] -- last state is '%s'", ma.State)
					return
				}

				if ma.ReplicateNotExecute(op.Database, op.Table) {
					ma.Logger.Debug("Execute[dmlupdate] -- replicate filter: skip")

					ma.metricCh <- MetricUnit{Name: MetricDestDMLUpdateSkip, Value: 1, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
					ma.metricCh <- MetricUnit{Name: MetricDestDMLInsertSkip, Value: 0, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
					ma.metricCh <- MetricUnit{Name: MetricDestDMLInsertSkip, Value: 0, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
					ma.metricCh <- MetricUnit{Name: MetricDestApplierOperationDMLUpdateSkip, Value: 1}
				} else {
					if gobRepairColumns, err := GobUint8NilRepair(op.AfterColumns); err != nil {
						ma.Logger.Error("Execute[dmlupdate] -- gob repair []uint8(nil): ", err)
						return
					} else {
						op.AfterColumns = gobRepairColumns
					}
					if gobRepairColumns, err := GobUint8NilRepair(op.BeforeColumns); err != nil {
						ma.Logger.Error("Execute[dmlupdate] -- gob repair []uint8(nil): ", err)
						return
					} else {
						op.BeforeColumns = gobRepairColumns
					}
					if err := ma.OnDMLUpdate(op); err != nil {
						return
					}
					ma.metricCh <- MetricUnit{Name: MetricDestDMLInsert, Value: 0, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
					ma.metricCh <- MetricUnit{Name: MetricDestDMLDelete, Value: 0, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
					ma.metricCh <- MetricUnit{Name: MetricDestDMLUpdate, Value: 1, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
					ma.metricCh <- MetricUnit{Name: MetricDestApplierOperationDMLUpdate, Value: 1}
				}

				ma.State = StateDML
			case MysqlOperationXid:
				ma.Logger.Debug("Operation[xid]")

				if !(ma.State == StateDML || ma.State == StateBEGIN) {
					ma.Logger.Error("Execute[xid] -- last state is '%s'", ma.State)
					return
				}

				ma.PendingCommitCount++

				ma.State = StateXID

				ma.metricCh <- MetricUnit{Name: MetricDestTrx, Value: 1}
			default:
				ma.Logger.Error("unknow operation.")
				return
			}

			ma.LastAppliedTimestamp = oper.GetTimestamp()

			ma.metricCh <- MetricUnit{Name: MetricDestApplierOperations, Value: 1}
			ma.metricCh <- MetricUnit{Name: MetricDestApplierTimestamp, Value: uint(ma.LastAppliedTimestamp)}
		}
	}
}

func (ma *MysqlApplier) OnDMLInsert(op MysqlOperationDMLInsert) error {
	ma.Logger.Debug("Execute[dmlinsert]")

	if ma.DmlModeInsert == "replace" {
		return ma.OnDMLInsert2(op)
	}

	return ma.OnDMLInsert1(op)
}

func (ma *MysqlApplier) OnDMLInsert1(op MysqlOperationDMLInsert) error {
	query, params := BuildDMLInsertQuery(op.Database, op.Table, op.Columns)
	ma.Logger.Trace("Execute[dmlinsert] -- Query: %s, Params: %#v", query, params)
	return ma.mysqlClient.ExecuteOnDML(query, params)
}

func (ma *MysqlApplier) OnDMLInsert2(op MysqlOperationDMLInsert) error {
	query, params := BuildDMLInsertQueryReplace(op.Database, op.Table, op.Columns)
	ma.Logger.Trace("Execute[dmlinsert] -- Query: %s, Params: %#v", query, params)
	return ma.mysqlClient.ExecuteOnDML(query, params)
}

func (ma *MysqlApplier) OnDMLDelete(op MysqlOperationDMLDelete) error {
	ma.Logger.Debug("Execute[dmldelete]")

	// todo
	if len(op.PrimaryKey) == 0 {
		ma.Logger.Warning("Execute[dmldelete] -- Not Primary Key -- SchemaContext: %s, Table: %s, Columne: %#v", op.Database, op.Table, op.Columns)
		return nil
	}

	query, params := BuildDMLDeleteQuery(op.Database, op.Table, op.Columns, op.PrimaryKey)
	ma.Logger.Trace("Execute[dmldelete] -- Query: %s, Params: %#v", query, params)
	return ma.mysqlClient.ExecuteOnDML(query, params)
}

func (ma *MysqlApplier) OnDMLUpdate(op MysqlOperationDMLUpdate) error {
	if ma.DmlModeUpdate == "replace" {
		return ma.OnDMLUpdate2(op)
	}

	return ma.OnDMLUpdate1(op)
}

func (ma *MysqlApplier) OnDMLUpdate1(op MysqlOperationDMLUpdate) error {
	// todo
	if len(op.PrimaryKey) == 0 {
		ma.Logger.Warning("Execute[dmlupdate] -- Not Primary Key -- SchemaContext: %s, Table: %s, BeforeColumne: %#v, AfterColume: %#v", op.Database, op.Table, op.BeforeColumns, op.AfterColumns)
		return nil
	}

	query, params := BuildDMLUpdateQuery(op.Database, op.Table, op.AfterColumns, op.BeforeColumns, op.PrimaryKey)
	ma.Logger.Trace("Execute[dmlupdate] -- Query: %s, Params: %#v", query, params)
	return ma.mysqlClient.ExecuteOnDML(query, params)
}

// delete first, then insert
func (ma *MysqlApplier) OnDMLUpdate2(op MysqlOperationDMLUpdate) error {
	// todo
	if len(op.PrimaryKey) == 0 {
		ma.Logger.Warning("Execute[dmlupdate] -- Not Primary Key -- SchemaContext: %s, Table: %s, BeforeColumne: %#v, AfterColume: %#v", op.Database, op.Table, op.BeforeColumns, op.AfterColumns)
		return nil
	}

	query, params := BuildDMLInsertQueryReplace(op.Database, op.Table, op.AfterColumns)
	ma.Logger.Trace("Execute[dmlupdate] -- Query: %s, Params: %#v", query, params)
	return ma.mysqlClient.ExecuteOnDML(query, params)
}

func (ma *MysqlApplier) OnDDLDatabase(op MysqlOperationDDLDatabase) error {
	return ma.mysqlClient.ExecuteOnNonDML("", op.Query)
}

func (ma *MysqlApplier) OnDDLTable(op MysqlOperationDDLTable) error {
	return ma.mysqlClient.ExecuteOnNonDML(op.SchemaContext, op.Query)
}

func (ma *MysqlApplier) OnDCLUser(op MysqlOperationDCLUser) error {
	return ma.mysqlClient.ExecuteOnNonDML(op.SchemaContext, op.Query)
}

func (ma *MysqlApplier) OnBegin(op MysqlOperationBegin) error {
	if err := ma.mysqlClient.Begin(); err != nil {
		return err
	}

	return nil
}

func (ma *MysqlApplier) OnGTID(op MysqlOperationGTID) error {
	// ma.Logger.Debug("OnGTID: %s:%d, lastcommitted: %d", op.ServerUUID, op.TrxID, op.LastCommitted)
	ma.GtidSkip = false
	if lastTrxID, ok := ma.ckpt.GetTrxIdOfServerUUID(op.ServerUUID); ok {
		if lastTrxID >= uint(op.TrxID) {
			ma.GtidSkip = true
			ma.Logger.Info("skip %s %d", op.ServerUUID, op.TrxID)
			return nil
		} else if uint(op.TrxID) >= lastTrxID+2 {
			err := fmt.Errorf("gtid miss '%s:%d'", op.ServerUUID, lastTrxID+1)
			ma.Logger.Error("OnGTID: %s.", err)
			return err
		}
	}

	if ma.GtidSkip {
		return nil
	}

	if ma.LastCommitted != op.LastCommitted {
		if err := ma.MergeCommit(); err != nil {
			return err
		}
		ma.LastCommitted = op.LastCommitted
	}

	ma.ckpt.SetTrxIdOfServerUUID(op.ServerUUID, uint(op.TrxID))

	ma.GtidCount++

	return nil
}

func (ma *MysqlApplier) EvaluateForceMergeCommit() bool {
	appliedCheckpointLag := ma.LastAppliedTimestamp - ma.LastCheckpointTimestamp //second

	// if ma.LastCommitted != op.LastCommitted {
	// 	ma.Logger.Debug("Execute[mergetrx] -- pending commit count: %d, gtid count: %d", ma.PendingCommitCount, ma.GtidCount)
	// 	return true
	// } else
	ma.Logger.Debug("Execute[mergetrx] -- force merge commit, pending commit count: %d over %d, gtid count: %d", ma.PendingCommitCount, ma.MergeCommitMaxCount, ma.GtidCount)
	ma.Logger.Debug("Execute[mergetrx] -- force merge commit, checkpoint delay: %ds over %ds, gtid count: %d", appliedCheckpointLag, ma.MergeCommitMaxDelay, ma.GtidCount)
	if ma.MergeCommitMaxCount >= 1 && ma.PendingCommitCount >= uint(ma.MergeCommitMaxCount) {
		ma.Logger.Debug("Execute[mergetrx] -- force merge commit, pending commit count: %d over %d, gtid count: %d", ma.PendingCommitCount, ma.MergeCommitMaxCount, ma.GtidCount)
		return true
	} else if ma.MergeCommitMaxDelay >= 1 && appliedCheckpointLag >= ma.MergeCommitMaxDelay {
		ma.Logger.Debug("Execute[mergetrx] -- force merge commit, checkpoint delay: %ds over %ds, gtid count: %d", appliedCheckpointLag, ma.MergeCommitMaxDelay, ma.GtidCount)
		return true
	}

	return false
}

func (ma *MysqlApplier) MergeCommit() error {
	if ma.PendingCommitCount >= 1 {
		if err := ma.mysqlClient.Commit(); err != nil {
			return err
		}

		ma.PendingCommitCount = 0
		ma.GtidCount = 0

		ma.metricCh <- MetricUnit{Name: MetricDestMergeTrx, Value: 1}

		ma.Checkpoint()
	}

	return nil
}

func (ma *MysqlApplier) Checkpoint() error {
	ma.Logger.Info("Checkpoint[gtid] -- gtidsets: %s", GetGtidSetsRangeStrFromGtidSetsMap(ma.ckpt.GtidSetsMap))
	ma.ckpt.PersistGtidSetsMaptToConsul()

	ma.Logger.Info("Checkpoint[binlogpos] -- binlogpos: %s:%d", ma.ckpt.BinLogFile, ma.ckpt.BinLogPos)
	ma.ckpt.PersistBinLogPosToConsul()

	ma.metricCh <- MetricUnit{Name: MetricDestCheckpointTimestamp, Value: uint(ma.LastCheckpointTimestamp)}

	return nil
}

func BuildDMLInsertQuery(datbaseName string, tableName string, columns []MysqlOperationDMLColumn) (string, []interface{}) {
	var keys []string
	var params []interface{}
	var placeholders []string

	for _, col := range columns {
		keys = append(keys, "`"+col.ColumnName+"`")
		params = append(params, col.ColumnValue)
		placeholders = append(placeholders, "?")
	}

	sql := fmt.Sprintf("INSERT INTO `%s`.`%s` (%s) VALUES (%s)", datbaseName, tableName, strings.Join(keys, ", "), strings.Join(placeholders, ", "))

	return sql, params
}

func BuildDMLInsertQueryReplace(datbaseName string, tableName string, columns []MysqlOperationDMLColumn) (string, []interface{}) {
	var keys []string
	var params []interface{}
	var placeholders []string

	for _, col := range columns {
		keys = append(keys, "`"+col.ColumnName+"`")
		params = append(params, col.ColumnValue)
		placeholders = append(placeholders, "?")
	}

	sql := fmt.Sprintf("REPLACE INTO `%s`.`%s` (%s) VALUES (%s)", datbaseName, tableName, strings.Join(keys, ", "), strings.Join(placeholders, ", "))

	return sql, params
}

func BuildDMLDeleteQuery(datbaseName string, tableName string, columns []MysqlOperationDMLColumn, primaryKey []uint64) (string, []interface{}) {
	wherePlaceholder, whereParams := GenerateConditionAndValues(primaryKey, columns)

	sql := fmt.Sprintf("DELETE FROM `%s`.`%s` WHERE %s", datbaseName, tableName, wherePlaceholder)

	return sql, whereParams
}

func BuildDMLUpdateQuery(datbaseName string, tableName string, afterColumns []MysqlOperationDMLColumn, beforeColumns []MysqlOperationDMLColumn, primaryKey []uint64) (string, []interface{}) {
	wherePlaceholder, whereParams := GenerateConditionAndValues(primaryKey, beforeColumns)

	var setClauses []string
	var setParams []interface{}

	for i, col := range afterColumns {
		setClauses = append(setClauses, fmt.Sprintf("`%s` = ?", col.ColumnName))
		setParams = append(setParams, afterColumns[i].ColumnValue)
	}

	sql := fmt.Sprintf("UPDATE `%s`.`%s` SET %s WHERE %s", datbaseName, tableName, strings.Join(setClauses, ", "), wherePlaceholder)

	return sql, append(setParams, whereParams...)
}

func GenerateConditionAndValues(primaryKeys []uint64, columns []MysqlOperationDMLColumn) (string, []interface{}) {
	var placeholder string
	var primary_values []interface{}

	if len(primaryKeys) >= 1 {
		parts := make([]string, len(primaryKeys))
		for i, k := range primaryKeys {
			parts[i] = fmt.Sprintf("`%s` = ?", columns[k].ColumnName)
			primary_values = append(primary_values, columns[k].ColumnValue)

		}
		placeholder = strings.Join(parts, " AND ")
	} else {
		parts := make([]string, len(columns))
		for i, c := range columns {
			parts[i] = fmt.Sprintf("`%s` = ?", c.ColumnName)
			primary_values = append(primary_values, c.ColumnValue)
		}
		placeholder = strings.Join(parts, " AND ")
		// for _, k := range primaryKeys {
		// 	primary_values = append(primary_values, columns[k].ColumnValue)
		// }
	}
	return placeholder, primary_values
}

// func BuildDMLUpdateQuery(datbaseName string, tableName string, columns []MysqlOperationDMLColumn, primaryKey []uint64) (string, []interface{}) {
// 	wherePlaceholder, whereParams := GenerateConditionAndValues(primaryKey, columns)

// 	setPlaceholder := make([]string, len(columns))
// 	var params []interface{}
// 	for i, c := range columns {
// 		setPlaceholder[i] = fmt.Sprintf("`%s` = ?", c.ColumnName)
// 		params = append(params, c.ColumnValue)
// 	}

// 	params = append(params, whereParams...)

// 	sql := fmt.Sprintf("UPDATE `%s`.`%s` SET %s WHERE %s", datbaseName, tableName, strings.Join(setPlaceholder, ", "), wherePlaceholder)

// 	return sql, params
// }

// func EscapeName(name string) string {
// 	if unquoted, err := strconv.Unquote(name); err == nil {
// 		name = unquoted
// 	}
// 	return fmt.Sprintf("`%s`", name)
// }

// func BuildDMLInsertQuery(databaseName, tableName string, tableColumns, sharedColumns, mappedSharedColumns *ColumnList, args []interface{}) (result string, sharedArgs []interface{}, err error) {
// 	if len(args) != tableColumns.Len() {
// 		return result, args, fmt.Errorf("args count differs from table column count in BuildDMLInsertQuery")
// 	}
// 	if !sharedColumns.IsSubsetOf(tableColumns) {
// 		return result, args, fmt.Errorf("shared columns is not a subset of table columns in BuildDMLInsertQuery")
// 	}
// 	if sharedColumns.Len() == 0 {
// 		return result, args, fmt.Errorf("No shared columns found in BuildDMLInsertQuery")
// 	}
// 	databaseName = EscapeName(databaseName)
// 	tableName = EscapeName(tableName)

// 	for _, column := range sharedColumns.Columns() {
// 		tableOrdinal := tableColumns.Ordinals[column.Name]
// 		arg := column.convertArg(args[tableOrdinal], false)
// 		sharedArgs = append(sharedArgs, arg)
// 	}

// 	mappedSharedColumnNames := duplicateNames(mappedSharedColumns.Names())
// 	for i := range mappedSharedColumnNames {
// 		mappedSharedColumnNames[i] = EscapeName(mappedSharedColumnNames[i])
// 	}
// 	preparedValues := buildColumnsPreparedValues(mappedSharedColumns)

// 	result = fmt.Sprintf(`
// 		replace /* mysql-sync %s.%s */
// 		into
// 			%s.%s
// 			(%s)
// 		values
// 			(%s)`,
// 		databaseName, tableName,
// 		databaseName, tableName,
// 		strings.Join(mappedSharedColumnNames, ", "),
// 		strings.Join(preparedValues, ", "),
// 	)
// 	return result, sharedArgs, nil
// }

// func BuildDMLDeleteQuery(databaseName, tableName string, tableColumns, uniqueKeyColumns *ColumnList, args []interface{}) (result string, uniqueKeyArgs []interface{}, err error) {
// 	if len(args) != tableColumns.Len() {
// 		return result, uniqueKeyArgs, fmt.Errorf("args count differs from table column count in BuildDMLDeleteQuery")
// 	}
// 	if uniqueKeyColumns.Len() == 0 {
// 		return result, uniqueKeyArgs, fmt.Errorf("No unique key columns found in BuildDMLDeleteQuery")
// 	}
// 	for _, column := range uniqueKeyColumns.Columns() {
// 		tableOrdinal := tableColumns.Ordinals[column.Name]
// 		arg := column.convertArg(args[tableOrdinal], true)
// 		uniqueKeyArgs = append(uniqueKeyArgs, arg)
// 	}
// 	databaseName = EscapeName(databaseName)
// 	tableName = EscapeName(tableName)
// 	equalsComparison, err := BuildEqualsPreparedComparison(uniqueKeyColumns.Names())
// 	if err != nil {
// 		return result, uniqueKeyArgs, err
// 	}
// 	result = fmt.Sprintf(`
// 		delete /* mysql-sync %s.%s */
// 		from
// 			%s.%s
// 		where
// 			%s`,
// 		databaseName, tableName,
// 		databaseName, tableName,
// 		equalsComparison,
// 	)
// 	return result, uniqueKeyArgs, nil
// }

// func (this *Applier) buildDMLEventQuery(dmlEvent *binlog.BinlogDMLEvent) (results [](*dmlBuildResult)) {
// 	switch dmlEvent.DML {
// 	case binlog.DeleteDML:
// 		{
// 			query, uniqueKeyArgs, err := sql.BuildDMLDeleteQuery(dmlEvent.DatabaseName, this.migrationContext.GetGhostTableName(), this.migrationContext.OriginalTableColumns, &this.migrationContext.UniqueKey.Columns, dmlEvent.WhereColumnValues.AbstractValues())
// 			return append(results, newDmlBuildResult(query, uniqueKeyArgs, -1, err))
// 		}
// 	case binlog.InsertDML:
// 		{
// 			query, sharedArgs, err := sql.BuildDMLInsertQuery(dmlEvent.DatabaseName, this.migrationContext.GetGhostTableName(), this.migrationContext.OriginalTableColumns, this.migrationContext.SharedColumns, this.migrationContext.MappedSharedColumns, dmlEvent.NewColumnValues.AbstractValues())
// 			return append(results, newDmlBuildResult(query, sharedArgs, 1, err))
// 		}
// 	case binlog.UpdateDML:
// 		{
// 			if _, isModified := this.updateModifiesUniqueKeyColumns(dmlEvent); isModified {
// 				dmlEvent.DML = binlog.DeleteDML
// 				results = append(results, this.buildDMLEventQuery(dmlEvent)...)
// 				dmlEvent.DML = binlog.InsertDML
// 				results = append(results, this.buildDMLEventQuery(dmlEvent)...)
// 				return results
// 			}
// 			query, sharedArgs, uniqueKeyArgs, err := sql.BuildDMLUpdateQuery(dmlEvent.DatabaseName, this.migrationContext.GetGhostTableName(), this.migrationContext.OriginalTableColumns, this.migrationContext.SharedColumns, this.migrationContext.MappedSharedColumns, &this.migrationContext.UniqueKey.Columns, dmlEvent.NewColumnValues.AbstractValues(), dmlEvent.WhereColumnValues.AbstractValues())
// 			args := sqlutils.Args()
// 			args = append(args, sharedArgs...)
// 			args = append(args, uniqueKeyArgs...)
// 			return append(results, newDmlBuildResult(query, args, 0, err))
// 		}
// 	}
// 	return append(results, newDmlBuildResultError(fmt.Errorf("Unknown dml event type: %+v", dmlEvent.DML)))
// }

// func BuildDMLUpdateQuery(databaseName, tableName string, tableColumns, sharedColumns, mappedSharedColumns, uniqueKeyColumns *ColumnList, valueArgs, whereArgs []interface{}) (result string, sharedArgs, uniqueKeyArgs []interface{}, err error) {
// 	if len(valueArgs) != tableColumns.Len() {
// 		return result, sharedArgs, uniqueKeyArgs, fmt.Errorf("value args count differs from table column count in BuildDMLUpdateQuery")
// 	}
// 	if len(whereArgs) != tableColumns.Len() {
// 		return result, sharedArgs, uniqueKeyArgs, fmt.Errorf("where args count differs from table column count in BuildDMLUpdateQuery")
// 	}
// 	if !sharedColumns.IsSubsetOf(tableColumns) {
// 		return result, sharedArgs, uniqueKeyArgs, fmt.Errorf("shared columns is not a subset of table columns in BuildDMLUpdateQuery")
// 	}
// 	if !uniqueKeyColumns.IsSubsetOf(sharedColumns) {
// 		return result, sharedArgs, uniqueKeyArgs, fmt.Errorf("unique key columns is not a subset of shared columns in BuildDMLUpdateQuery")
// 	}
// 	if sharedColumns.Len() == 0 {
// 		return result, sharedArgs, uniqueKeyArgs, fmt.Errorf("No shared columns found in BuildDMLUpdateQuery")
// 	}
// 	if uniqueKeyColumns.Len() == 0 {
// 		return result, sharedArgs, uniqueKeyArgs, fmt.Errorf("No unique key columns found in BuildDMLUpdateQuery")
// 	}
// 	databaseName = EscapeName(databaseName)
// 	tableName = EscapeName(tableName)

// 	for _, column := range sharedColumns.Columns() {
// 		tableOrdinal := tableColumns.Ordinals[column.Name]
// 		arg := column.convertArg(valueArgs[tableOrdinal], false)
// 		sharedArgs = append(sharedArgs, arg)
// 	}

// 	for _, column := range uniqueKeyColumns.Columns() {
// 		tableOrdinal := tableColumns.Ordinals[column.Name]
// 		arg := column.convertArg(whereArgs[tableOrdinal], true)
// 		uniqueKeyArgs = append(uniqueKeyArgs, arg)
// 	}

// 	setClause, err := BuildSetPreparedClause(mappedSharedColumns)
// 	if err != nil {
// 		return "", sharedArgs, uniqueKeyArgs, err
// 	}

// 	equalsComparison, err := BuildEqualsPreparedComparison(uniqueKeyColumns.Names())
// 	if err != nil {
// 		return "", sharedArgs, uniqueKeyArgs, err
// 	}
// 	result = fmt.Sprintf(`
// 		update /* mysql-sync %s.%s */
// 			%s.%s
// 		set
// 			%s
// 		where
// 			%s`,
// 		databaseName, tableName,
// 		databaseName, tableName,
// 		setClause,
// 		equalsComparison,
// 	)
// 	return result, sharedArgs, uniqueKeyArgs, nil
// }
