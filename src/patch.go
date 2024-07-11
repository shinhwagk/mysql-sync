package main

import "fmt"

// important
// []uint8{} -> gob -> []uint8(nil)
// 252 == tinyblob tinytext blob text mediumblob mediumtext longblob longtext
func gobUint8NilRepair(columns []MysqlOperationDMLColumn) ([]MysqlOperationDMLColumn, error) {
	for _, col := range columns {
		if col.ColumnType == 252 && !col.ColumnValueIsNil {
			if colVal, ok := col.ColumnValue.([]byte); ok && colVal == nil {
				col.ColumnValue = []uint8{}
			} else if !ok {
				errMsg := fmt.Errorf("gob repair: %s %d %#v %#v %#v %#v %#v", col.ColumnName, col.ColumnType, col.ColumnValue, col.ColumnValueIsNil, col.ColumnValue == nil, col.ColumnType == 252, !col.ColumnValueIsNil)
				return nil, errMsg
			}
		}
	}
	return columns, nil
}
