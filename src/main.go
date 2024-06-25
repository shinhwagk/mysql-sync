package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

func main() {
	logger := NewLogger(1, "main")

	config, err := LoadConfig("/etc/mysqlsync/config.yml")
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to load config: %v", err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repl := flag.Bool("repl", false, "Start replication.")
	dest := flag.Bool("dest", false, "Start destination.")
	destName := flag.String("name", "", "Name for the destination.")

	flag.Parse()

	if *repl && *dest {
		fmt.Println("Error: --repl and --dest cannot be used together.")
		os.Exit(1)
	}

	if *repl {
		replication := NewReplication(*config)
		if err := replication.start(ctx, cancel); err != nil {
			logger.Error("Replication start error: " + err.Error())
			cancel()
		}
	} else if *dest {
		if *destName == "" {
			fmt.Println("Error: 'name' parameter is required when 'dest' is specified.")
			return
		}
		replName := config.Replication.Name
		destConfig := config.Destinations[0]
		hjdbAddr := config.HJDB.Addr
		destination := NewDestination(replName, destConfig, hjdbAddr)
		if err := destination.Start(ctx, cancel); err != nil {
			logger.Error("Destination start error: " + err.Error())
			cancel()
		}
	} else {
		fmt.Println("No specific mode activated")

	}
}
