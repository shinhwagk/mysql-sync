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

	flag.Parse()

	if *repl && *dest {
		fmt.Println("Error: --repl and --dest cannot be used together.")
		os.Exit(1)
	}

	if *repl {
		replication := NewReplication(config.Name, config.Replication)
		if err := replication.start(ctx, cancel); err != nil {
			logger.Error("Replication start error: " + err.Error())
			cancel()
		}
	} else if *dest {
		destination := NewDestination(config.Name, config.Destination)
		if err := destination.Start(ctx, cancel); err != nil {
			logger.Error("Destination start error: " + err.Error())
			cancel()
		}
	} else {
		fmt.Println("No specific mode activated")
	}
}
