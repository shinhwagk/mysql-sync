package main

import (
	"context"
	"fmt"

	"github.com/go-mysql-org/go-mysql/client"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
)

// Create a binlog syncer with a unique server id, the server id must be different from other MySQL's.
// flavor is mysql or mariadb
func xxx(c *client.Conn) error {
	c.SetCapability(mysql.CLIENT_ZSTD_COMPRESSION_ALGORITHM)
	c.Compression = mysql.MYSQL_COMPRESS_ZSTD
	return nil
}
func main() {
	// conn, _ := client.Connect("127.0.0.1:3306", "root", "", "test")

	// cfg := replication.BinlogSyncerConfig{
	// 	ServerID: 100,
	// 	Flavor:   "mysql",
	// 	Host:     "db1",
	// 	Port:     3306,
	// 	User:     "root",
	// 	Password: "root_password",
	// 	Option:   xxx,
	// }
	cfg := replication.BinlogSyncerConfig{
		ServerID: 100,
		Flavor:   "mysql",
		Host:     "172.16.0.179",
		Port:     3306,
		User:     "ghost",
		Password: "54448hotINBOX",
		Option:   xxx,
	}
	syncer := replication.NewBinlogSyncer(cfg)

	// streamer, _ := syncer.StartSync(mysql.Position{"mysql-bin.000001", 4})

	gset, _ := mysql.ParseGTIDSet("mysql", "73b24aef-0b4d-11ef-9a54-1418774ca835:1-29611485,eb559d55-f6e4-11ee-94e4-c81f66d988c2:1-52461618")
	// gset, _ := mysql.ParseGTIDSet("mysql", "027634d7-10d1-11ef-a658-0242ac170004:1-2")
	streamer, _ := syncer.StartSyncGTID(gset)

	// or you can start a gtid replication like
	// streamer, _ := syncer.StartSyncGTID(gtidSet)
	// the mysql GTID set likes this "de278ad0-2106-11e4-9f8e-6edd0ca20947:1-2"
	// the mariadb GTID set likes this "0-1-100"

	for {
		ev, _ := streamer.GetEvent(context.Background())
		if ev.Header.EventType == replication.GTID_EVENT {
			wrEvent := ev.Event.(*replication.GTIDEvent)
			gtid, _ := wrEvent.GTIDNext()
			fmt.Println(gtid)
		} else if ev.Header.EventType == replication.TRANSACTION_PAYLOAD_EVENT {
			wrEvent := ev.Event.(*replication.TransactionPayloadEvent)

			fmt.Println("xxxxx", wrEvent.CompressionType)
		}

		// Dump event
	}

}
