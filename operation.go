package main

import (
	"fmt"
	"strings"
)

type MysqlOperation interface {
	Statement()
}

type MysqlOperationDDLTable struct {
	schema string
	table  string
	query  string
}

func (op MysqlOperationDDLTable) Statement() {

}

func (op MysqlOperationBegin) Statement() {

}

type OperationDDLDatabase struct {
	schema string
	query  string
}

func (op OperationDDLDatabase) Statement() {

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
	PrimaryKey interface{}
}

func (op MysqlOperationDMLInsert) Statement() {

}

type MysqlOperationDMLDelete struct {
	Database   string
	Table      string
	Columns    []MysqlOperationDMLColumn
	PrimaryKey interface{}
}

func (op MysqlOperationDMLDelete) Statement() {

}

type MysqlOperationDMLUpdate struct {
	Database      string
	Table         string
	AfterColumns  []MysqlOperationDMLColumn
	BeforeColumns []MysqlOperationDMLColumn
	PrimaryKey    interface{}
}

func (op MysqlOperationDMLUpdate) Statement() {

}

type OperationDML interface {
	GenerateSQL() (string, []interface{})
}

func (op *MysqlOperationDMLInsert) GenerateSQL() (string, []interface{}) {
	fmt.Println("GenerateSQL")
	var keys []string
	var params []interface{}
	var placeholders []string

	for _, col := range op.Columns {
		keys = append(keys, col.ColumnName)
		params = append(params, col.ColumnValue)
		placeholders = append(placeholders, "?")
	}

	sql := fmt.Sprintf("REPLACE INTO `%s`.`%s` (%s) VALUES (%s)", op.Database, op.Table, strings.Join(keys, ", "), strings.Join(placeholders, ", "))
	fmt.Println("Insert GenerateSQL", sql, params)
	return sql, params
}

// GenerateSQL generates the SQL statement and parameters for DELETE operations
func (op *MysqlOperationDMLDelete) GenerateSQL() (string, []interface{}) {
	condition := fmt.Sprintf("%s = ?", op.PrimaryKey)
	sql := fmt.Sprintf("DELETE FROM `%s`.`%s` WHERE %s", op.Database, op.Table, condition)
	params := []interface{}{op.PrimaryKey}
	return sql, params
}

// GenerateSQL generates the SQL statement and parameters for UPDATE operations
// func (op *MysqlOperationDMLUpdate) GenerateSQL() (string, []interface{}) {
// 	var sets []string
// 	var params []interface{}

// 	for k, v := range op.AfterColumns {
// 		sets = append(sets, fmt.Sprintf("%s = ?", k))
// 		params = append(params, v)
// 	}

// 	condition := fmt.Sprintf("%s = ?", op.PrimaryKey)
// 	params = append(params, op.BeforeValues[op.PrimaryKey])

// 	sql := fmt.Sprintf("UPDATE `%s`.`%s` SET %s WHERE %s",
// 		op.Schema, op.Table, strings.Join(sets, ", "), condition)

// 	return sql, params
// }

type MysqlOperationXid struct {
	Timestamp uint32
}

func (op MysqlOperationXid) Statement() {

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

func (op MysqlOperationGTID) Statement() {

}

type MysqlOperationHeartbeat struct {
	Timestamp uint32
}

func (op MysqlOperationHeartbeat) Statement() {

}
