package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
)

// Create a binlog syncer with a unique server id, the server id must be different from other MySQL's.
// flavor is mysql or mariadb

func main() {
	cfg := replication.BinlogSyncerConfig{
		ServerID: 100,
		Flavor:   "mysql",
		Host:     "db1",
		Port:     3306,
		User:     "root",
		Password: "root_passowrd",
	}
	syncer := replication.NewBinlogSyncer(cfg)
	fmt.Println("xxxxx")

	// Start sync with specified binlog file and position
	gtidSet, err := mysql.ParseGTIDSet("mysql", "d185500e-225e-11ef-8161-0242ac140006:1-172")

	if err != nil {
		fmt.Println("sdfsfsdfsd", err)
	}

	streamer, _ := syncer.StartSyncGTID(gtidSet)

	for {
		fmt.Println("xxxxx1")

		ev, err := streamer.GetEvent(context.Background())
		if err != nil {
			fmt.Println(err.Error())
		}

		// Dump event
		ev.Dump(os.Stdout)
		// time.Sleep(time.Second * 60)
	}
}
