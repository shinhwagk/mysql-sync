package main

import (
	"fmt"
	"strconv"
	"strings"

	"canal/canal"

	"github.com/go-mysql-org/go-mysql/schema"
)

type MysqlSyncSource struct {
	canal.DummyEventHandler
}

func escapeValue(value interface{}, column schema.TableColumn) string {
	if strings.Contains(column.RawType, "int") || strings.Contains(column.RawType, "decimal") || strings.Contains(column.RawType, "double") || strings.Contains(column.RawType, "float") {
		return fmt.Sprintf("%v", value)
	} else if strings.Contains(column.RawType, "date") || strings.Contains(column.RawType, "time") {
		return fmt.Sprintf("'%v'", value)
	} else if strings.HasPrefix(column.RawType, "set") {
		fmt.Println("set", value, column.SetValues)

		strValue, ok := value.(string)
		if !ok {
			fmt.Println("")
		}
		intValue, err := strconv.Atoi(strValue)
		if err != nil {
			fmt.Println("")
		}
		x := column.SetValues[intValue]
		return fmt.Sprintf("'%s'", strings.Replace(fmt.Sprintf("%s", x), "'", "''", -1))
	} else if strings.HasPrefix(column.RawType, "enum") {
		fmt.Println(column, value, column.EnumValues)

	} else {
		return ""
	}
	return ""
}

func buildInsertSQL(e *canal.RowsEvent) string {
	var cols []string
	var vals []interface{}
	// e.Table.Columns[0].
	for _, col := range e.Table.Columns {
		cols = append(cols, col.Name)
	}
	for _, row := range e.Rows {
		var valueStrings []string
		for i, val := range row {
			valueStrings = append(valueStrings, escapeValue(val, e.Table.Columns[i]))
		}
		vals = append(vals, fmt.Sprintf("(%s)", strings.Join(valueStrings, ", ")))
	}
	return fmt.Sprintf("INSERT INTO %s.%s (%s) VALUES %s;", e.Table.Schema, e.Table.Name, strings.Join(cols, ", "), "")
}
