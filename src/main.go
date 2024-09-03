package main

import (
	"context"
	"flag"
	"os"
)

func main() {
	logger := NewLogger(1, "main")

	config, err := LoadConfig("/etc/mysqlsync/config.yml")
	if err != nil {
		logger.Error("Failed to load config: %v", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repl := flag.Bool("repl", false, "Start replication.")
	dest := flag.Bool("dest", false, "Start destination.")
	destName := flag.String("name", "", "Name for the destination.")

	flag.Parse()

	if *repl && *dest {
		logger.Error("--repl and --dest cannot be used together.")
	} else if *repl {
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
	} else {
		flag.Usage()
	}
	os.Exit(1)
}
