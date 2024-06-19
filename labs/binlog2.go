package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"

	_ "github.com/pingcap/tidb/pkg/types/parser_driver"
)

func main() {
	// 假设这是配置和创建 BinlogSyncer 的代码段
	config := replication.BinlogSyncerConfig{
		ServerID: 1001,
		Flavor:   "mysql",
		Host:     "db1",
		Port:     3306,
		User:     "root",
		Password: "root_password",
	}

	syncer := replication.NewBinlogSyncer(config)
	streamer, err := syncer.StartSync(mysql.Position{"mysql-bin.000001", 0})

	if err != nil {
		log.Fatal("Failed to start sync:", err)
	}

	if streamer == nil {
		log.Fatal("Streamer is nil")
	}
	childCtx, childCancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(time.Second * 5)
		// binlogExtract.Stop()
		childCancel()
		fmt.Println("binlogext stop")
	}()
	fmt.Println(syncer, streamer, 2222)

	cnt := 1
	for {
		fmt.Println("GetEvent GetEventGetEventGetEventGetEventGetEvent")
		_, err := streamer.GetEvent(childCtx)
		if err != nil {
			if err == context.Canceled {
				fmt.Println("context.Canceled", err)
				syncer.Close()
				fmt.Println(syncer, streamer, 2222)
				time.Sleep(time.Second * 2)
				fmt.Println(syncer, streamer, 2222)
				if syncer == nil {
					fmt.Println("333333333333")
				}
				fmt.Println(2222, err.Error())

				break
			}
			fmt.Println("sdsssss", err.Error())
		}
		cnt += 1
		fmt.Println(cnt)

		// switch e := ev.Event.(type) {
		// case *replication.QueryEvent:

		// 	// fmt.Println(string(e.Query))
		// }

		// 处理 binlog 事件
		// log.Printf("Event: %v", ev.Header.EventType)
	}
}
