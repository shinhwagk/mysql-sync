package main

import (
	"fmt"
	"reflect"

	"github.com/go-mysql-org/go-mysql/mysql"
)

func printType(value interface{}) {
	switch v := value.(type) {
	case int:
		fmt.Printf("Integer: %d\n", v)
	case float64:
		fmt.Printf("Float64: %f\n", v)
	case string:
		fmt.Printf("String: %s\n", v)
	case bool:
		fmt.Printf("Boolean: %t\n", v)
	default:
		fmt.Printf("Unsupported type: %T\n", v)
	}
}

// mysql.MYSQL_TYPE_BLOB longtext []uint8
// mysql.MYSQL_TYPE_BLOB tinytext []uint8
// mysql.MYSQL_TYPE_DATETIME2 datetime string
// mysql.MYSQL_TYPE_DATE date string
// mysql.MYSQL_TYPE_TINY tinyint int8
// mysql.MYSQL_TYPE_STRING char string
// mysql.MYSQL_TYPE_VARCHAR varchar string
// mysql.MYSQL_TYPE_LONG int int32
// mysql.MYSQL_TYPE_TIMESTAMP2 timestamp string
// mysql.MYSQL_TYPE_STRING set int64
// mysql.MYSQL_TYPE_STRING enum int64

func columnTypeAstrict(colName string, colType byte, colValue interface{}) (string, error) {
	switch colType {
	case mysql.MYSQL_TYPE_NULL:
	}
	// case mysql.MYSQL_TYPE_TIMESTAMP2:
	// 	if reflect.TypeOf(colValue).Kind() == reflect.String {
	// 		return "timestamp", nil
	// 	}
	// case mysql.MYSQL_TYPE_DATETIME2:
	// 	if colValue == nil || reflect.TypeOf(colValue).Kind() == reflect.String {
	// 		return "datetime", nil
	// 	}
	// case mysql.MYSQL_TYPE_DATE:
	// 	if colValue == nil || reflect.TypeOf(colValue).Kind() == reflect.String {
	// 		return "date", nil
	// 	}
	// case mysql.MYSQL_TYPE_TINY:
	// 	if colValue == nil || reflect.TypeOf(colValue).Kind() == reflect.Int8 {
	// 		return "tinyint", nil
	// 	}
	// case mysql.MYSQL_TYPE_LONG:
	// 	if colValue == nil || reflect.TypeOf(colValue).Kind() == reflect.Int32 {
	// 		return "smallint", nil
	// 	}
	// case mysql.MYSQL_TYPE_STRING:
	// 	if colValue == nil || reflect.TypeOf(colValue).Kind() == reflect.String {
	// 		return "char", nil
	// 	}
	// case mysql.MYSQL_TYPE_VARCHAR:
	// 	if colValue == nil || reflect.TypeOf(colValue).Kind() == reflect.String {
	// 		return "varchar", nil
	// 	}
	// case mysql.MYSQL_TYPE_BLOB:
	// 	if colValue == nil || reflect.TypeOf(colValue).Elem().Kind() == reflect.Uint8 {
	// 		return "tinytext", nil
	// 	}
	// default:
	// 	return "", fmt.Errorf("column type unprocess %d %s ", colType, reflect.TypeOf(colValue))
	// }
	return "", fmt.Errorf("column type unmatch  %s %d %s ", colName, colType, reflect.TypeOf(colValue))
}

func emptyChannel(ch <-chan interface{}) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}
