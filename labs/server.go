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
	listener, err := net.Listen("tcp", "localhost:9999")
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		return
	}
	defer listener.Close()
	fmt.Println("Listening on localhost:9999")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting:", err.Error())
			return
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// 创建一个示例数据对象
	object := LargeDataObject{
		Name:    "Example",
		Details: "Contains binary data",
		Data:    []byte{1, 2, 3, 4, 5}, // 示例二进制数据
	}

	// 创建 zlib 压缩 writer
	zw := zlib.NewWriter(conn)
	defer zw.Close()

	// 创建并使用 gob encoder
	encoder := gob.NewEncoder(zw)
	if err := encoder.Encode(object); err != nil {
		fmt.Println("Encode error:", err)
	}
}
