package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/go-mysql-org/go-mysql/mysql"
)

func PrintType(value interface{}) {
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

func ParseDSN(dsn string) (user, password, host, port string, err error) {
	atSplit := strings.Split(dsn, "@tcp(")
	if len(atSplit) != 2 {
		err = fmt.Errorf("invalid DSN format")
		return
	}

	userPass := strings.Split(atSplit[0], ":")
	if len(userPass) != 2 {
		err = fmt.Errorf("invalid user:password format")
		return
	}
	user = userPass[0]
	password = userPass[1]

	hostPort := strings.TrimSuffix(atSplit[1], ")/")
	hostPortSplit := strings.Split(hostPort, ":")
	if len(hostPortSplit) != 2 {
		err = fmt.Errorf("invalid host:port format")
		return
	}
	host = hostPortSplit[0]
	port = hostPortSplit[1]

	return
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

func splitAndClean(s *string) []string {
	if s == nil {
		return nil
	}
	parts := strings.Split(*s, ",")
	var results []string
	for _, part := range parts {
		cleaned := strings.TrimSpace(part)
		if cleaned != "" {
			results = append(results, cleaned)
		}
	}
	return results
}

//	func contains(arr []string, str string) bool {
//		for _, item := range arr {
//			if item == str {
//				return true
//			}
//		}
//		return false
//	}
func contains(value string, list []string) bool {
	for _, v := range list {
		if strings.TrimSpace(v) == value {
			return true
		}
	}
	return false
}

func updateSlice(slice []int, newItem int) []int {
	for i, value := range slice {
		if value == 0 {
			slice[i] = newItem
		}
	}
	slice = append(slice[1:], newItem)
	return slice
}

func calculateAdjustedMean(slice []int) int {
	sliceCopy := make([]int, len(slice))
	copy(sliceCopy, slice)

	sort.Ints(sliceCopy)
	sliceCopy = sliceCopy[1 : len(slice)-1]

	total := 0
	for _, value := range sliceCopy {
		total += value
	}

	return total / len(sliceCopy)
}
