package main

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
	_ "github.com/go-sql-driver/mysql"

	_ "github.com/pingcap/tidb/pkg/types/parser_driver"
)

func columnTypeAstrict(colName string, colType byte, colValue interface{}) {
	fmt.Println("===========")
	if reflect.TypeOf(colValue).Kind() == reflect.Slice {
		fmt.Println("2", colName, colType, "slice", reflect.TypeOf(colValue).Elem().Kind())
	} else {
		fmt.Println("2", colName, colType, reflect.TypeOf(colValue).Kind())
	}
	switch colType {
	case mysql.MYSQL_TYPE_TIMESTAMP2:
		if reflect.TypeOf(colValue).Kind() == reflect.String {
			fmt.Println("timestamp", reflect.TypeOf(colValue).Kind())
		}
	case mysql.MYSQL_TYPE_DATETIME2:
		if colValue == nil || reflect.TypeOf(colValue).Kind() == reflect.String {
			fmt.Println("datetime", reflect.TypeOf(colValue).Kind())
		}
	case mysql.MYSQL_TYPE_DATE:
		if colValue == nil || reflect.TypeOf(colValue).Kind() == reflect.String {
			fmt.Println("date", reflect.TypeOf(colValue).Kind())
		}
	case mysql.MYSQL_TYPE_TINY:
		if colValue == nil || reflect.TypeOf(colValue).Kind() == reflect.Int8 {
			fmt.Println("tinyint", reflect.TypeOf(colValue).Kind())
		}
	case mysql.MYSQL_TYPE_LONG:
		if colValue == nil || reflect.TypeOf(colValue).Kind() == reflect.Int32 {
			fmt.Println("smallint", reflect.TypeOf(colValue).Kind())
		}
	case mysql.MYSQL_TYPE_STRING:
		if colValue == nil || reflect.TypeOf(colValue).Kind() == reflect.String {
			fmt.Println("char", reflect.TypeOf(colValue).Kind())
		}
	case mysql.MYSQL_TYPE_VARCHAR:
		if colValue == nil || reflect.TypeOf(colValue).Kind() == reflect.String {
			fmt.Println("varchar", reflect.TypeOf(colValue).Kind())
		}
	case mysql.MYSQL_TYPE_BLOB:
		if colValue == nil || reflect.TypeOf(colValue).Elem().Kind() == reflect.Uint8 {
			fmt.Println("tinytext", reflect.TypeOf(colValue).Elem().Kind())
		}
	default:
		fmt.Println("column type unprocess %d %s ", colType, reflect.TypeOf(colValue))
	}
	// return "", fmt.Errorf("column type unmatch  %s %d %s ", colName, colType, reflect.TypeOf(colValue))
}

func main() {
	
	// 假设这是配置和创建 BinlogSyncer 的代码段
	config := replication.BinlogSyncerConfig{
		ServerID: 1001,
		Flavor:   "mysql",
		Host:     "db1",
		Port:     3306,
		User:     "root",
		Password: "root_password",
	}

	syncer := replication.NewBinlogSyncer(config)
	gtidSet, _ := mysql.ParseGTIDSet("mysql", "73b24aef-0b4d-11ef-9a54-1418774ca835:1-93998679,eb559d55-f6e4-11ee-94e4-c81f66d988c2:1-52461618")

	streamer, err := syncer.StartSyncGTID(gtidSet)

	if err != nil {
		log.Fatal("Failed to start sync:", err)
	}

	for {
		ev, err := streamer.GetEvent(context.Background())
		if err != nil {
			log.Fatal("Error getting event:", err)
			break
		}
		// a := []byte(nil)

		switch e := ev.Event.(type) {
		case *replication.GTIDEvent:
			if gtidNext, err := e.GTIDNext(); err != nil {
				fmt.Println(err.Error())
			} else {
				parts := strings.Split(gtidNext.String(), ":")
				if len(parts) == 2 {
					xid, err := strconv.ParseInt(parts[1], 10, 64)
					if err != nil {
						fmt.Println(err.Error())
					} else {
						fmt.Println(parts[0], xid)
					}
				}
			}
		case *replication.RowsEvent:
			if string(e.Table.Table) == "test" {
				continue
			}
			// fmt.Println("1111111", ev.Header.EventType, replication.UPDATE_ROWS_EVENTv0, replication.UPDATE_ROWS_EVENTv1, replication.UPDATE_ROWS_EVENTv2)
			fmt.Println("2", string(e.Table.Schema), string(e.Table.Table), len(e.Rows))
			for j := 0; j < len(e.Rows); j += 1 {

				for i := 0; i < int(e.ColumnCount); i += 1 {
					ct := e.Table.ColumnType[i]
					cn := string(e.Table.ColumnName[i])

					columnTypeAstrict(cn, ct, e.Rows[j][i])
				}
				// before_value := e.Rows[i]
				// // after_value := e.Rows[i+1]

				// for i, a := range before_value {
				// 	fmt.Printf("Param %d: Type: %T, Value: %v, IsNil: %v, IsEmptyByteSlice: %v\n", i, a, a, a == nil, reflect.DeepEqual(a, []uint8(nil)))
				// }

				// for i, a := range after_value {
				// 	fmt.Printf("Param %d: Type: %T, Value: %v, IsNil: %v, IsEmptyByteSlice: %v\n", i, a, a, a == nil, reflect.DeepEqual(a, []uint8(nil)))
				// }

			}
		}
	}
}
