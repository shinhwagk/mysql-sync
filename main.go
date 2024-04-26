package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func processBinlogEntry(binlogEntry string) string {
	if strings.HasPrefix(binlogEntry, "#") {
		if strings.HasPrefix(binlogEntry, "# at ") {
		} else if strings.HasPrefix(binlogEntry, "#2") {
			parts := strings.Fields(binlogEntry)
			if parts[9] == "Write_rows" {
			}

		} else if strings.HasPrefix(binlogEntry, "#700101  ") {
		}
		return ""
	} else {
		if strings.HasPrefix(binlogEntry, "SET @@SESSION.GTID_NEXT= '") && strings.HasSuffix(binlogEntry, "'/*!*/;") {
		} else if binlogEntry == "COMMIT/*!*/;" || binlogEntry == "ROLLBACK/*!*/;" {
		} else if strings.HasPrefix(binlogEntry, "SET TIMESTAMP=") && strings.HasSuffix(binlogEntry, "'/*!*/;") {
		} else if strings.HasPrefix(binlogEntry, "BINLOG '") {
		} else {
		}
		return binlogEntry + "\n"
	}
}

func main() {

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {

		line := scanner.Text()

		processedLine := processBinlogEntry(line)

		if processedLine != "" {
			os.Stdout.Write([]byte(processedLine))
		}

	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "读取标准输入时出错:", err)
	}

}
