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
	StateBEGIN = "begin"
	StateXID   = "xid"
	StateGTID  = "gtid"
)

func NewMysqlApplier(logLevel int, ckpt *Checkpoint, mysqlClient *MysqlClient, replicate *Replicate, metricCh chan<- MetricUnit) *MysqlApplier {
	return &MysqlApplier{
		Logger:                  NewLogger(logLevel, "mysql-applier"),
		ckpt:                    ckpt,
		mysqlClient:             mysqlClient,
		replicate:               replicate,
		LastCommitted:           0,
		LastCheckpointTimestamp: 0,
		AllowCommit:             false,
		metricCh:                metricCh,
		GtidSkip:                false,
		State:                   StateNULL,
	}
}

type MysqlApplier struct {
	Logger                  *Logger
	mysqlClient             *MysqlClient
	replicate               *Replicate
	ckpt                    *Checkpoint
	LastCommitted           int64
	LastCheckpointTimestamp uint32
	AllowCommit             bool
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
			if ma.GtidSkip {
				if _, ok := oper.(MysqlOperationGTID); !ok {
					ma.metricCh <- MetricUnit{Name: MetricDestApplierTimestamp, Value: uint(oper.GetTimestamp())}
					continue
				}
			}

			switch op := oper.(type) {
			case MysqlOperationDDLDatabase:
				ma.LastCheckpointTimestamp = op.Timestamp
				if ma.State == StateGTID {
					ma.State = StateDDL

					ma.metricCh <- MetricUnit{Name: MetricDestApplierOperationDDLDatabase, Value: 1}

					if ma.ReplicateNotExecute(op.Schema, "") {
						ma.metricCh <- MetricUnit{Name: MetricDestDDLDatabaseSkip, Value: 1, LabelPair: map[string]string{"database": op.Schema}}
					} else {
						// Submit dml before ddl execution
						if err := ma.MergeCommit(); err != nil {
							return
						}
						if err := ma.OnDDLDatabase(op); err != nil {
							return
						}
						ma.metricCh <- MetricUnit{Name: MetricDestDDLDatabase, Value: 1, LabelPair: map[string]string{"database": op.Schema}}
					}
					ma.Checkpoint()
				} else {
					ma.Logger.Error("execute DDL(database): last state is '%s'", ma.State)
					return
				}
			case MysqlOperationDDLTable:
				ma.LastCheckpointTimestamp = op.Timestamp
				if ma.State == StateGTID {
					ma.State = StateDDL

					ma.metricCh <- MetricUnit{Name: MetricDestApplierOperationDDLTable, Value: 1}

					if ma.ReplicateNotExecute(op.Schema, op.Table) {
						ma.metricCh <- MetricUnit{Name: MetricDestDDLTableSkip, Value: 1, LabelPair: map[string]string{"database": op.Schema, "table": op.Table}}
					} else {
						if err := ma.MergeCommit(); err != nil {
							return
						}
						if err := ma.OnDDLTable(op); err != nil {
							return
						}
						ma.metricCh <- MetricUnit{Name: MetricDestDDLTable, Value: 1, LabelPair: map[string]string{"database": op.Schema, "table": op.Table}}
					}
					ma.Checkpoint()
				} else {
					ma.Logger.Error("execute DDL(table): last state is '%s'", ma.State)
					return
				}
			case MysqlOperationDMLInsert:
				if ma.State == StateBEGIN || ma.State == StateDML {
					ma.State = StateDML

					ma.metricCh <- MetricUnit{Name: MetricDestApplierOperationDMLInsert, Value: 1}

					if ma.ReplicateNotExecute(op.Database, op.Table) {
						ma.metricCh <- MetricUnit{Name: MetricDestDMLInsertSkip, Value: 1, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
						ma.metricCh <- MetricUnit{Name: MetricDestDMLUpdateSkip, Value: 0, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
						ma.metricCh <- MetricUnit{Name: MetricDestDMLDeleteSkip, Value: 0, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}

					} else {
						if gobRepairColumns, err := GobUint8NilRepair(op.Columns); err != nil {
							ma.Logger.Error("gob repair []uint8(nil): ", err)
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
					}
				} else {
					ma.Logger.Error("execute DML: last state is '%s'", ma.State)
					return
				}
			case MysqlOperationDMLDelete:
				if ma.State == StateBEGIN || ma.State == StateDML {
					ma.State = StateDML

					ma.metricCh <- MetricUnit{Name: MetricDestApplierOperationDMLDelete, Value: 1}

					if ma.ReplicateNotExecute(op.Database, op.Table) {
						ma.metricCh <- MetricUnit{Name: MetricDestDMLDeleteSkip, Value: 1, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
						ma.metricCh <- MetricUnit{Name: MetricDestDMLUpdateSkip, Value: 0, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
						ma.metricCh <- MetricUnit{Name: MetricDestDMLInsertSkip, Value: 0, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
					} else {
						if gobRepairColumns, err := GobUint8NilRepair(op.Columns); err != nil {
							ma.Logger.Error("gob repair []uint8(nil): ", err)
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
					}
				} else {
					ma.Logger.Error("execute DML: last state is '%s'", ma.State)
					return
				}
			case MysqlOperationDMLUpdate:
				if ma.State == StateBEGIN || ma.State == StateDML {
					ma.State = StateDML

					ma.metricCh <- MetricUnit{Name: MetricDestApplierOperationDMLUpdate, Value: 1}

					if ma.ReplicateNotExecute(op.Database, op.Table) {
						ma.metricCh <- MetricUnit{Name: MetricDestDMLUpdateSkip, Value: 1, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
						ma.metricCh <- MetricUnit{Name: MetricDestDMLInsertSkip, Value: 0, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}
						ma.metricCh <- MetricUnit{Name: MetricDestDMLInsertSkip, Value: 0, LabelPair: map[string]string{"database": op.Database, "table": op.Table}}

					} else {
						if gobRepairColumns, err := GobUint8NilRepair(op.AfterColumns); err != nil {
							ma.Logger.Error("gob repair []uint8(nil): ", err)
							return
						} else {
							op.AfterColumns = gobRepairColumns
						}
						if gobRepairColumns, err := GobUint8NilRepair(op.BeforeColumns); err != nil {
							ma.Logger.Error("gob repair []uint8(nil): ", err)
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
					}
				} else {
					ma.Logger.Error("execute DML: last state is '%s'", ma.State)
					return
				}
			case MysqlOperationXid:
				ma.LastCheckpointTimestamp = op.Timestamp
				if ma.State == StateDML || ma.State == StateBEGIN {
					ma.State = StateXID

					ma.metricCh <- MetricUnit{Name: MetricDestApplierOperationXid, Value: 1}

					ma.AllowCommit = true
					if err := ma.OnXID(op); err != nil {
						return
					}
					ma.metricCh <- MetricUnit{Name: MetricDestTrx, Value: 1}

				} else {
					ma.Logger.Error("event Xid: last state is '%s'", ma.State)
					return
				}
			case MysqlOperationGTID:
				if ma.State == StateXID || ma.State == StateDDL || ma.State == StateNULL {
					ma.State = StateGTID

					ma.metricCh <- MetricUnit{Name: MetricDestApplierOperationGtid, Value: 1}

					if err := ma.OnGTID(op); err != nil {
						return
					}
				} else {
					ma.Logger.Error("operation(gtid): last state is '%s'", ma.State)
					return
				}
			case MysqlOperationHeartbeat:
				ma.metricCh <- MetricUnit{Name: MetricDestApplierOperationHeartbeat, Value: 1}
				if err := ma.OnHeartbeat(op); err != nil {
					return
				}
				ma.State = StateNULL
				ma.metricCh <- MetricUnit{Name: MetricDestCheckpointTimestamp, Value: uint(op.GetTimestamp())}
			case MysqlOperationBegin:
				if ma.State == StateGTID {
					ma.State = StateBEGIN

					ma.metricCh <- MetricUnit{Name: MetricDestApplierOperationBegin, Value: 1}

					if !ma.AllowCommit {
						if err := ma.OnBegin(op); err != nil {
							return
						}
					} else {
						ma.Logger.Debug("Skip begin because merge commit.")
					}
				} else {
					ma.Logger.Error("operation(begin): last state is '%s'", ma.State)
					return
				}
			case MysqlOperationBinLogPos:
				ma.metricCh <- MetricUnit{Name: MetricDestApplierOperationBinLogPos, Value: 1}
				ma.ckpt.SetBinlogPos(op.File, op.Pos)
			default:
				ma.Logger.Error("unknow operation.")
				return
			}

			ma.metricCh <- MetricUnit{Name: MetricDestApplierTimestamp, Value: uint(oper.GetTimestamp())}
		}
	}
}

func (ma *MysqlApplier) OnDMLInsert(op MysqlOperationDMLInsert) error {
	query, params := BuildDMLInsertQuery(op.Database, op.Table, op.Columns)
	ma.Logger.Debug("OnDMLInsert -- SchemaContext: %s, Table: %s", op.Database, op.Table)
	ma.Logger.Trace("OnDMLInsert -- SchemaContext: %s, Table: %s, Query: %s, Params: %v", op.Database, op.Table, query, params)
	if err := ma.mysqlClient.ExecuteDML(query, params); err != nil {
		return err
	}
	return nil
}

func (ma *MysqlApplier) OnDMLDelete(op MysqlOperationDMLDelete) error {
	// todo
	if len(op.PrimaryKey) == 0 {
		ma.Logger.Warning("OnDMLDelete -- Not Primarykey -- SchemaContext: %s, Table: %s, Columne: %v", op.Database, op.Table, op.Columns)
		return nil
	}

	query, params := BuildDMLDeleteQuery(op.Database, op.Table, op.Columns, op.PrimaryKey)
	ma.Logger.Debug("OnDMLDelete -- SchemaContext: %s, Table: %s", op.Database, op.Table)
	ma.Logger.Trace("OnDMLDelete -- SchemaContext: %s, Table: %s, Query: %s, Params: %v", op.Database, op.Table, query, params)
	if err := ma.mysqlClient.ExecuteDML(query, params); err != nil {
		return err
	}

	return nil
}

func (ma *MysqlApplier) OnDMLUpdate(op MysqlOperationDMLUpdate) error {
	// todo
	if len(op.PrimaryKey) == 0 {
		ma.Logger.Warning("OnDMLUpdate -- Not Primarykey -- SchemaContext: %s, Table: %s, BeforeColumne: %v, AfterColume: %v", op.Database, op.Table, op.BeforeColumns, op.AfterColumns)
		return nil
	}

	ma.Logger.Debug("OnDMLUpdate -- SchemaContext: %s, Table: %s", op.Database, op.Table)

	query, params := BuildDMLDeleteQuery(op.Database, op.Table, op.BeforeColumns, op.PrimaryKey)
	ma.Logger.Trace("OnDMLUpdate -- Delete -- SchemaContext: %s, Table: %s, Query: %s, Params: %v", op.Database, op.Table, query, params)
	if err := ma.mysqlClient.ExecuteDML(query, params); err != nil {
		return err
	}

	query, params = BuildDMLInsertQuery(op.Database, op.Table, op.AfterColumns)
	ma.Logger.Trace("OnDMLUpdate -- Insert -- SchemaContext: %s, Table: %s, Query: %s, Params: %v", op.Database, op.Table, query, params)
	if err := ma.mysqlClient.ExecuteDML(query, params); err != nil {
		return err
	}

	return nil
}

func (ma *MysqlApplier) OnDDLDatabase(op MysqlOperationDDLDatabase) error {
	ma.Logger.Debug("OnDDLDatabase -- query: '%s'", op.Query)

	if err := ma.mysqlClient.ExecuteOnDatabase(op.Query); err != nil {
		ma.Logger.Error("OnDDLDatabase: %s.", err)
		return err
	}

	return nil
}

func (ma *MysqlApplier) OnDDLTable(op MysqlOperationDDLTable) error {
	ma.Logger.Debug("OnDDLTable -- SchemaContext: %s, Database: %s, Query: %s", op.SchemaContext, op.Schema, op.Query)

	if err := ma.mysqlClient.ExecuteOnTable(op.SchemaContext, op.Query); err != nil {
		ma.Logger.Error("OnDDLTable: %s.", err)
		return err
	}

	return nil
}

func (ma *MysqlApplier) OnXID(op MysqlOperationXid) error {
	ma.Logger.Debug("OnXID")

	return nil
}

func (ma *MysqlApplier) OnBegin(op MysqlOperationBegin) error {
	ma.Logger.Debug("OnBegin")

	if err := ma.mysqlClient.Begin(); err != nil {
		return err
	}

	return nil
}

func (ma *MysqlApplier) OnGTID(op MysqlOperationGTID) error {
	ma.Logger.Debug("OnGTID: %s:%d, lastcommitted: %d", op.ServerUUID, op.TrxID, op.LastCommitted)
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

	if ma.LastCommitted != op.LastCommitted {
		if err := ma.MergeCommit(); err != nil {
			return err
		}
		ma.LastCommitted = op.LastCommitted
	}

	ma.ckpt.SetTrxIdOfServerUUID(op.ServerUUID, uint(op.TrxID))

	return nil
}

func (ma *MysqlApplier) OnHeartbeat(op MysqlOperationHeartbeat) error {
	ma.Logger.Debug("OnHeartbeat")
	if err := ma.MergeCommit(); err != nil {
		return err
	}
	return nil
}

func (ma *MysqlApplier) MergeCommit() error {
	if ma.AllowCommit {
		if err := ma.mysqlClient.Commit(); err != nil {
			ma.Logger.Error("Merge commit: %", err)
			return err
		}
		ma.Logger.Debug("Merge commit complate.")

		ma.Checkpoint()

		ma.metricCh <- MetricUnit{Name: MetricDestMergeTrx, Value: 1}
		ma.AllowCommit = false
	}

	return nil
}

func (ma *MysqlApplier) Checkpoint() error {
	if err := ma.ckpt.PersistGtidSetsMaptToConsul(); err == nil {
		ma.Logger.Info("Checkpoint GTID: %s", GetGtidSetsRangeStrFromGtidSetsMap(ma.ckpt.GtidSetsMap))
	}

	if err := ma.ckpt.PersistBinLogPosToConsul(); err == nil {
		ma.Logger.Info("Checkpoint BINLOGPOS: %s:%d", ma.ckpt.BinLogFile, ma.ckpt.BinLogPos)
	}

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

	sql := fmt.Sprintf("REPLACE INTO `%s`.`%s` (%s) VALUES (%s)", datbaseName, tableName, strings.Join(keys, ", "), strings.Join(placeholders, ", "))

	return sql, params
}

func BuildDMLDeleteQuery(datbaseName string, tableName string, columns []MysqlOperationDMLColumn, primaryKey []uint64) (string, []interface{}) {
	wherePlaceholder, whereParams := GenerateConditionAndValues(primaryKey, columns)

	sql := fmt.Sprintf("DELETE FROM `%s`.`%s` WHERE %s", datbaseName, tableName, wherePlaceholder)

	return sql, whereParams
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
