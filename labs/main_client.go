package main

import (
	"encoding/gob"
	"fmt"
	"net"
	"time"

	"github.com/klauspost/compress/zstd"
)

type Message interface {
	MsgType() string
}

type TextMessage struct {
	Text string
}

func (m TextMessage) MsgType() string { return "text" }

type ImageMessage struct {
	URL string
}

func (m ImageMessage) MsgType() string { return "image" }

type WrappedMessage struct {
	Type    string
	Message Message
}

func init() {
	gob.Register(&TextMessage{})
	gob.Register(&ImageMessage{})
	// gob.Register(&WrappedMessage{})
}

func main() {
	conn, err := net.Dial("tcp", "localhost:9999")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	zstdReader, err := zstd.NewReader(conn)
	if err != nil {
		fmt.Println("Error creating zstd reader:", err)
		return
	}
	defer zstdReader.Close()

	decoder := gob.NewDecoder(zstdReader)

	// decoder := gob.NewDecoder(conn)

	for {
		var wrappedMsg []Message
		if err := decoder.Decode(&wrappedMsg); err != nil {
			fmt.Println("Error decoding message:", err)
			break
		}
		fmt.Println("sdfsfsdf %s", len(wrappedMsg))

		time.Sleep(time.Second * 81000)
		// fmt.Println("Received message type:", wrappedMsg.MsgType())
		// // 处理解码后的消息
		// switch msg := wrappedMsg.(type) {
		// case *TextMessage:
		// 	fmt.Printf("Text Message: %s\n", msg.Text)
		// case *ImageMessage:
		// 	fmt.Printf("Image Message: %s\n", msg.URL)
		// default:
		// 	fmt.Println("Unknown message type received")
		// }
	}
}
