package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func NewMysqlApplier(logLevel int, hjdb *HJDB, mysqlClient *MysqlClient, metricCh chan<- MetricUnit) *MysqlApplier {
	return &MysqlApplier{
		Logger: NewLogger(logLevel, "mysql applier"),

		mysqlClient: mysqlClient,
		hjdb:        hjdb,

		GtidSets:               make(map[string]uint),
		LastGtidSets:           make(map[string]uint),
		LastServerUUID:         "",
		LastCommitted:          0,
		LastOperationTimestamp: 0,
		AllowCommit:            false,

		// metric:   &MetricDestination{0, 0, 0, 0, 0, 0, 0, 0},
		MetricDelay: 0,
		metricCh:    metricCh,
	}
}

type MysqlApplier struct {
	Logger      *Logger
	mysqlClient *MysqlClient
	hjdb        *HJDB

	// state
	GtidSets               map[string]uint
	LastGtidSets           map[string]uint
	LastServerUUID         string
	LastCommitted          int64
	LastOperationTimestamp uint32
	AllowCommit            bool

	MetricDelay uint
	metricCh    chan<- MetricUnit

	// metric *MetricDestination
}

func (ma *MysqlApplier) Start(ctx context.Context, moCh <-chan MysqlOperation) {
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			timer.Reset(100 * time.Millisecond)
		case <-ctx.Done():
			ma.Logger.Info("ctx done signal received.")
			if err := ma.mysqlClient.Rollback(); err != nil {
				ma.mysqlClient.Logger.Info("mysql connect rollback error: " + err.Error())
			} else {
				ma.mysqlClient.Logger.Info("mysql connect rollback complate.")
			}
			if err := ma.mysqlClient.Close(); err != nil {
				ma.mysqlClient.Logger.Info("mysql connect close error: " + err.Error())
			} else {
				ma.mysqlClient.Logger.Info("mysql connect close complate.")
			}
			return
		case oper, ok := <-moCh:
			if !ok {
				ma.Logger.Info("mysql operation channel closed.")
				return
			}
			ma.metricCh <- MetricUnit{Name: MetricApplierOperations, Value: 1}
			switch op := oper.(type) {
			case MysqlOperationDDLDatabase:
				if err := ma.OnDDLDatabase(op); err != nil {
					ma.Logger.Error(fmt.Sprintf("OnDDLDatabase -- %s", err))
					return
				}
				ma.metricCh <- MetricUnit{Name: MetricDestDMLInsertTimes, Value: 1}
			case MysqlOperationDMLInsert:
				if err := ma.OnDMLInsert(op); err != nil {
					ma.Logger.Error("MysqlOperationDMLInsert " + err.Error())
					return
				}
				ma.metricCh <- MetricUnit{Name: MetricDestDMLInsertTimes, Value: 1}
			case MysqlOperationDMLDelete:
				if err := ma.OnDMLDelete(op); err != nil {
					ma.Logger.Error("MysqlOperationDMLDelete " + err.Error())
					return
				}
				ma.metricCh <- MetricUnit{Name: MetricDestDMLDeleteTimes, Value: 1}
			case MysqlOperationDMLUpdate:
				if err := ma.OnDMLUpdate(op); err != nil {
					ma.Logger.Error("MysqlOperationDMLUpdate " + err.Error())
					return
				}
				ma.metricCh <- MetricUnit{Name: MetricDestDMLUpdateTimes, Value: 1}
			case MysqlOperationDDLTable:
				if err := ma.OnDDLTable(op); err != nil {
					ma.Logger.Error("MysqlOperationDDLTable " + err.Error())
					return
				}
			case MysqlOperationXid:
				if err := ma.OnXID(op); err != nil {
					ma.Logger.Error("MysqlOperationXid " + err.Error())
					return
				}
				ma.metricCh <- MetricUnit{Name: MetricDestTrx, Value: 1}
			case MysqlOperationGTID:
				if err := ma.OnGTID(op); err != nil {
					ma.Logger.Error("MysqlOperationGTID " + err.Error())
					return
				}
			case MysqlOperationHeartbeat:
				if err := ma.OnHeartbeat(op); err != nil {
					ma.Logger.Error("MysqlOperationHeartbeat " + err.Error())
					return
				}
			case MysqlOperationBegin:
				if err := ma.OnBegin(op); err != nil {
					ma.Logger.Error("MysqlOperationBegin " + err.Error())
					return
				}
			default:
				fmt.Println("xxxx")
			}
			timer.Reset(100 * time.Millisecond)
		}
	}
}

func (ma *MysqlApplier) OnDMLInsert(op MysqlOperationDMLInsert) error {
	sql, params := BuildDMLInsertQuery(op.Database, op.Table, op.Columns)

	// ma.logger.Debug(fmt.Sprintf("OnDMLInsert -- query: '%s' params: '%s'", sql, params))

	if err := ma.mysqlClient.ExecuteDML(sql, params); err != nil {
		return err
	}
	return nil
}

func (ma *MysqlApplier) OnDMLDelete(op MysqlOperationDMLDelete) error {
	sql, params := BuildDMLDeleteQuery(op.Database, op.Table, op.Columns, op.PrimaryKey)

	// ma.logger.Debug(fmt.Sprintf("OnDMLDelete -- query: '%s' params: '%s'", sql, params))

	if err := ma.mysqlClient.ExecuteDML(sql, params); err != nil {
		return err
	}

	return nil
}

func (ma *MysqlApplier) OnDMLUpdate(op MysqlOperationDMLUpdate) error {
	sql, params := BuildDMLDeleteQuery(op.Database, op.Table, op.BeforeColumns, op.PrimaryKey)
	ma.Logger.Debug(fmt.Sprintf("OnDMLUpdate -- query: '%s' params: '%s'", sql, params))
	if err := ma.mysqlClient.ExecuteDML(sql, params); err != nil {
		return err
	}

	sql, params = BuildDMLInsertQuery(op.Database, op.Table, op.AfterColumns)
	ma.Logger.Debug(fmt.Sprintf("OnDMLUpdate -- query: '%s' params: '%s'", sql, params))
	if err := ma.mysqlClient.ExecuteDML(sql, params); err != nil {
		return err
	}

	return nil
}

func (ma *MysqlApplier) OnDDLDatabase(op MysqlOperationDDLDatabase) error {
	ma.Logger.Debug(fmt.Sprintf("OnDDLDatabase -- query: '%s'", op.Query))

	ma.LastOperationTimestamp = op.Timestamp

	if err := ma.mysqlClient.ExecuteOnDatabase(op.Query); err != nil {
		return err
	}

	return ma.Checkpoint()
}

func (ma *MysqlApplier) OnDDLTable(op MysqlOperationDDLTable) error {
	ma.Logger.Debug(fmt.Sprintf("OnDDLTable -- schema: %s  query: %s", op.Schema, op.Query))

	ma.LastOperationTimestamp = op.Timestamp

	if err := ma.mysqlClient.ExecuteOnTable(op.Schema, op.Query); err != nil {
		ma.Logger.Error(fmt.Sprintf("OnDDLTable -- %s", err))
		return err
	}

	return ma.Checkpoint()
}

func (ma *MysqlApplier) OnXID(op MysqlOperationXid) error {
	ma.LastOperationTimestamp = op.Timestamp
	ma.AllowCommit = true

	return nil
}

func (ma *MysqlApplier) OnBegin(op MysqlOperationBegin) error {
	// ma.logger.Debug(fmt.Sprintf("OnBegin"))

	if err := ma.mysqlClient.Begin(); err != nil {
		return err
	}

	return nil
}

func (ma *MysqlApplier) OnGTID(op MysqlOperationGTID) error {
	ma.Logger.Debug(fmt.Sprintf("OnGTID -- %s:%d", op.ServerUUID, op.TrxID))

	if ma.LastCommitted != op.LastCommitted {
		ma.MergeCommit()
		ma.LastCommitted = op.LastCommitted
	}
	ma.LastGtidSets[op.ServerUUID] = uint(op.TrxID)
	ma.LastServerUUID = op.ServerUUID
	return nil
}

func (ma *MysqlApplier) OnHeartbeat(op MysqlOperationHeartbeat) error {
	ma.LastOperationTimestamp = op.Timestamp
	err := ma.MergeCommit()

	ma.metricCh <- MetricUnit{Name: MetricDestDelay, Value: uint(time.Now().Unix() - int64(op.Timestamp))}
	return err
}

func (ma *MysqlApplier) MergeCommit() error {
	if ma.AllowCommit {
		ma.AllowCommit = false

		if err := ma.mysqlClient.Commit(); err != nil {
			return err
		}

		ma.Logger.Debug(fmt.Sprintf("MergeCommit -- %T", ma.LastGtidSets))
		ma.metricCh <- MetricUnit{Name: MetricDestMergeTrx, Value: 1}

		return ma.Checkpoint()
	}

	return nil
}

func (ma *MysqlApplier) Checkpoint() error {
	// ma.Logger.Info(fmt.Sprintf("Checkpoint -- %s:%d", ma.LastServerUUID, ma.LastGtidSets[ma.LastServerUUID]))

	ma.metricCh <- MetricUnit{Name: MetricDestDelay, Value: uint(time.Now().Unix() - int64(ma.LastOperationTimestamp))}

	if err := ma.hjdb.Update("file", "gtidset", ma.LastGtidSets); err != nil {
		ma.Logger.Error(fmt.Sprintf("Checkpoint -- %s", err))
		return err
	}

	return nil
}

func BuildDMLInsertQuery(datbaseName string, tableName string, columns []MysqlOperationDMLColumn) (string, []interface{}) {
	var keys []string
	var params []interface{}
	var placeholders []string

	for _, col := range columns {
		keys = append(keys, col.ColumnName)
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

// func BuildDMLUpdateQuery() (string, []interface{}) {
// 	params := []interface{}{}

// 	wherePlaceholder, whereParams := GenerateConditionAndValues(op.PrimaryKey, op.BeforeColumns)

// 	setPlaceholder := make([]string, len(op.AfterColumns))

// 	for i, c := range op.AfterColumns {
// 		setPlaceholder[i] = fmt.Sprintf("`%s` = ?", c.ColumnName)
// 		params = append(params, c.ColumnValue)
// 	}

// 	for _, param := range whereParams {
// 		params = append(params, param)
// 	}

// 	sql := fmt.Sprintf("UPDATE `%s`.`%s` SET %s WHERE %s", op.Database, op.Table, strings.Join(setPlaceholder, ", "), wherePlaceholder)

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
