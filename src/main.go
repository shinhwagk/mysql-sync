package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

func main() {
	logger := NewLogger(1, "main")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	configPath := flag.String("config", "", "Path to config file.")
	repl := flag.Bool("repl", false, "Start replication.")
	dest := flag.Bool("dest", false, "Start destination.")
	destName := flag.String("name", "", "Name for the destination (requires --dest).")

	flag.Usage = func() {
		fmt.Println("================================================================")

		fmt.Println("Usage of mysqlsync:")
		fmt.Println()
		fmt.Println("  0. Configuration file :")
		fmt.Println("       https://github.com/shinhwagk/mysql-sync/blob/main/src/config-testing.yml")
		fmt.Println()
		fmt.Println("----------------------------------------------------------------")

		fmt.Println("  1. Replication mode command:")
		fmt.Println("       mysqlsync -config config.yml -repl")
		fmt.Println()
		fmt.Println("  2. Destination mode command:")
		fmt.Println("       mysqlsync -config config.yml -dest -name dest1")
		fmt.Println("================================================================")
		os.Exit(1)
	}

	flag.Parse()

	if (*repl && *dest) || (!*repl && !*dest) || (*dest && *destName == "") {
		fmt.Println()
		fmt.Println("================================================================")
		fmt.Println("Error: Invalid parameter configuration. Please check your input.")
		flag.Usage()
	}

	config, err := LoadConfig(*configPath)
	if err != nil {
		logger.Error("Failed to load config: %v", err)
		os.Exit(1)
	}

	if *repl {
		replication := NewReplication(*config)
		replication.Start(ctx, cancel)
		return
	} else if *dest {
		if *destName == "" {
			logger.Error("'name' parameter is required when 'dest' is specified.")
		} else {
			destination := NewDestination(*config, *destName)
			destination.Start(ctx, cancel)
			return
		}
	}
	os.Exit(1)
}
