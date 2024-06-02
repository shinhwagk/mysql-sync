package main

import (
	"compress/zlib"
	"encoding/gob"
	"fmt"
	"net"
)

type LargeDataObject struct {
	Name    string
	Details string
	Data    []byte
}

func main() {
	conn, err := net.Dial("tcp", "localhost:9999")
	if err != nil {
		fmt.Println("Error dialing:", err.Error())
		return
	}
	defer conn.Close()

	// 创建 zlib 解压缩 reader
	zr, err := zlib.NewReader(conn)
	if err != nil {
		fmt.Println("Error creating zlib reader:", err)
		return
	}
	defer zr.Close()

	// 使用 gob decoder 解码数据
	var object LargeDataObject
	decoder := gob.NewDecoder(zr)
	if err := decoder.Decode(&object); err != nil {
		fmt.Println("Decode error:", err)
		return
	}

	fmt.Printf("Received: %+v\n", object)
}
