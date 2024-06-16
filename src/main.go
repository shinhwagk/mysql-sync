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

	repl := flag.Bool("destination", false, "Activate REPL mode")
	dest := flag.Bool("replication", false, "Specify the destination")

	// 解析命令行参数
	flag.Parse()

	if *repl && *dest {
		fmt.Println("Error: --repl and --dest cannot be used together.")
		os.Exit(1)
	}

	if *repl {
		replication := NewReplication(config.Name, config.Replication)
		replication.start(ctx, cancel)
	} else if *dest {
		destination := NewDestination(config.Name, config.Destination)
		destination.Start(ctx, cancel)
	} else {
		fmt.Println("No specific mode activated")
	}

}
