package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/klauspost/compress/zstd"
)

func init() {
	gob.Register(&MysqlOperationHeartbeat{})
	gob.Register(&MysqlOperationGTID{})
	gob.Register(&MysqlOperationDDLDatabase{})

	gob.Register(&DTOPause{})
	gob.Register(&DTOResume{})
	gob.Register(&DTOGtidSet{})

}

// func sendServer(encoder *gob.Encoder, dto DTO) {
// 	if err := encoder.Encode(&dto); err != nil {
// 		fmt.Println("Error encoding DTO1:", err)
// 		return
// 	}

// }
func main() {
	conn, err := net.Dial("tcp", "localhost:9998")
	if err != nil {
		log.Fatal("Dial error:", err)
	}
	defer conn.Close()

	zstdReader, err := zstd.NewReader(conn)
	if err != nil {
		fmt.Println("Error creating zstd reader:", err)
		return
	}
	defer zstdReader.Close()

	decoder := gob.NewDecoder(zstdReader)
	encoder := gob.NewEncoder(conn)

	// ctrlCh := make(chan string)

	ch := make(chan MysqlOperation, 10)

	x := 3

	// go func() {
	// 	for cmd := range ctrlCh {
	// 		_, err := conn.Write([]byte(cmd + "\n"))
	// 		fmt.Println("send pause")
	// 		if err != nil {
	// 			fmt.Println("Error sending control command:", err)
	// 			break
	// 		}
	// 	}
	// }()

	go func() {
		for oper := range ch {
			fmt.Println("process oper", oper)
			time.Sleep(time.Second * time.Duration(x))
			x--
			if x == 0 && x <= 5 {
				x++
			}
		}
	}()

	var cache_operations []MysqlOperation = []MysqlOperation{}

	for {
		var operations []MysqlOperation
		if err := decoder.Decode(&operations); err != nil {
			fmt.Println("Error decoding message:", err)
			break
		}

		for _, oper := range operations {
			select {
			case ch <- oper:
			default:
				sendServer(encoder, &DTOPause{})
				// ctrlCh <- "pause"
				fmt.Println("cache ooper", len(cache_operations))
				cache_operations = append(cache_operations, oper)
			}
		}

		if len(cache_operations) >= 1 {
			for len(cache_operations) > 0 {
				select {
				case ch <- cache_operations[0]:
					cache_operations = cache_operations[1:]
				}
			}
			if len(cache_operations) == 0 {
				// ctrlCh <- "resume"
				sendServer(encoder, &DTOResume{})
				fmt.Println("send resume")
			}
		}
	}
}
