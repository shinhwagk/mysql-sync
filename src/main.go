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
		logger.Error("--repl and --dest cannot be used together.")
		os.Exit(1)
	}

	if *repl {
		replication := NewReplication(*config)
		replication.start(ctx, cancel)
		cancel()
	} else if *dest {
		if *destName == "" {
			logger.Error("'name' parameter is required when 'dest' is specified.")
			return
		}

		destination := NewDestination(*config, *destName)
		if err := destination.Start(ctx, cancel); err != nil {
			logger.Error("Destination start: " + err.Error())
			cancel()
		}
	} else {
		logger.Info("No specific mode activated")
	}
}
