package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
)

type RegexHandler struct {
	Regex   *regexp.Regexp
	Handler func(string)
}

var binlogEntryAnalyzer = []RegexHandler{
	{Regex: regexp.MustCompile(`^# at [0-9]+$`), Handler: func(line string) { fmt.Printf("Handling regex2 for line: %s\n", line) }},
	{Regex: regexp.MustCompile(`^SET TIMESTAMP=[0-9]{10}(?:\.[0-9]{6})?/\*!*\*/;$`), Handler: func(line string) { fmt.Printf("Handling regex1 for line: %s\n", line) }},
	{Regex: regexp.MustCompile(`^SET @@SESSION\.GTID_NEXT= '([0-9a-z]{8}-[0-9a-z]{4}-[0-9a-z]{4}-[0-9a-z]{4}-[0-9a-z]{12}:[0-9]+)'`), Handler: func(line string) { fmt.Printf("Handling regex3 for line: %s\n", line) }},
	{Regex: regexp.MustCompile(`^COMMIT/\*!*\*/;$`), Handler: func(line string) { fmt.Printf("Handling regex3 for line: %s\n", line) }},
	{Regex: regexp.MustCompile(`^ROLLBACK/\*!*\*/;$`), Handler: func(line string) { fmt.Printf("Handling regex3 for line: %s\n", line) }},
	{Regex: regexp.MustCompile(` Rotate to `), Handler: func(line string) { fmt.Printf("Handling regex3 for line: %s\n", line) }},
}

func processLine(line string) {
	for _, entry := range binlogEntryAnalyzer {
		if entry.Regex.MatchString(line) {
			entry.Handler(line)
			break
		}
	}
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		
		processLine(line)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "error reading standard input: %v\n", err)
	}
}
