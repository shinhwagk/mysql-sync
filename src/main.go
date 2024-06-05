package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-mysql-org/go-mysql/replication"
)

func main() {
	config, err := LoadConfig("/workspaces/mysqlbinlog-sync/src/config.yml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Println(config)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(20 * time.Second)

		cancel()
	}()

	ch := make(chan MysqlOperation)

	logger := &Logger{LevelDebug}

	hjdb := NewHJDB(config.HJDB.Addr, config.HJDB.DB, logger)

	metricDirector := &MetricDirector{hjdb, logger}

	fmt.Println(config.Replication.ServerID)
	cfg := replication.BinlogSyncerConfig{
		ServerID:        uint32(config.Replication.ServerID),
		Flavor:          "mysql",
		Host:            config.Replication.Host,
		Port:            uint16(config.Replication.Port),
		User:            config.Replication.User,
		Password:        config.Destination.Password,
		Charset:         "utf8mb4",
		HeartbeatPeriod: time.Second * 1,
	}

	canal := NewCanal(ctx, cancel, cfg, config.Replication.GTID, ch, logger)

	//"root:root_password@tcp(db2:3306)/sys?autocommit=false"
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?%s", config.Destination.User, config.Replication.Password, config.Replication.Host, config.Destination.Port, config.Destination.Params)
	fmt.Println(dsn)

	md1 := NewMysqlDestination(ctx, cancel, logger, config.Destination.Name, ch, dsn, hjdb, metricDirector)

	canal.AddDestination(md1)

	canal.Run()
}
