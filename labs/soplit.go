package main

import (
	"fmt"
	"strings"
)

func splitAndClean(s string) []string {
	parts := strings.Split(s, ",")
	var results []string
	for _, part := range parts {
		cleaned := strings.TrimSpace(part)
		if cleaned != "" {
			results = append(results, cleaned)
		}
	}
	return results
}

func main() {
	fmt.Println("Encoded URL:", splitAndClean(""))
}
