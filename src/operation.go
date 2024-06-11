package main

import (
	"fmt"
	"strings"
)

type MysqlOperation interface {
	OperationType() string
}
type MysqlOperationDDLTable struct {
	Schema    string
	Table     string
	Query     string
	Timestamp uint32
}

func (op MysqlOperationDDLTable) OperationType() string {
	return "MysqlOperationDDLTable"
}

func (op MysqlOperationBegin) OperationType() string {
	return "MysqlOperationBegin"

}

type MysqlOperationDDLDatabase struct {
	Schema    string
	Query     string
	Timestamp uint32
}

func (op MysqlOperationDDLDatabase) OperationType() string {
	return "MysqlOperationDDLDatabase"

}

type MysqlOperationDMLColumn struct {
	ColumnName  string
	ColumnType  byte
	ColumnValue interface{}
}

type MysqlOperationDMLInsert struct {
	Database   string
	Table      string
	Columns    []MysqlOperationDMLColumn
	PrimaryKey []uint64
}

func (op MysqlOperationDMLInsert) OperationType() string {
	return "MysqlOperationDMLInsert"
}

type MysqlOperationDMLDelete struct {
	Database   string
	Table      string
	Columns    []MysqlOperationDMLColumn
	PrimaryKey []uint64
}

func (op MysqlOperationDMLDelete) OperationType() string {
	return "MysqlOperationDMLDelete"
}

type MysqlOperationDMLUpdate struct {
	Database      string
	Table         string
	AfterColumns  []MysqlOperationDMLColumn
	BeforeColumns []MysqlOperationDMLColumn
	PrimaryKey    []uint64
}

func (op MysqlOperationDMLUpdate) OperationType() string {
	return "MysqlOperationDMLUpdate"
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

type MysqlOperationXid struct {
	Timestamp uint32
}

func (op MysqlOperationXid) OperationType() string {
	return "MysqlOperationXid"

}

type MysqlOperationGTID struct {
	LastCommitted int64
	ServerID      uint32
	Timestamp     uint32
	ServerUUID    string
	TrxID         int64
	// GtidNext      string
}

type MysqlOperationBegin struct {
	ServerID  uint32
	Timestamp uint32
}

func (op MysqlOperationGTID) OperationType() string {
	return "MysqlOperationGTID"
}

type MysqlOperationHeartbeat struct {
	Timestamp uint32
}

func (op MysqlOperationHeartbeat) OperationType() string {
	return "MysqlOperationHeartbeat"
}
