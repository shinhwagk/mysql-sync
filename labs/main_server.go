package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
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
	listener, err := net.Listen("tcp", ":9999")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	fmt.Println("Server start.")
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn)
	}
}

type Abc struct {
	megs []Message
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	// countingWriter := &CountingWriter{Conn: conn, BytesWritten: 0}

	var buf bytes.Buffer

	multiWriter := io.MultiWriter(conn, &buf)

	// zlibWriter := zlib.NewWriter(tee)
	// defer zlibWriter.Close()

	zstdWriter, err := zstd.NewWriter(multiWriter)
	if err != nil {
		fmt.Println("Error creating zstd writer:", err)
		return
	}
	defer zstdWriter.Close()

	encoder := gob.NewEncoder(zstdWriter)

	// encoder := gob.NewEncoder(conn)

	for {
		buf.Reset() // Reset buffer before each write
		messages := []Message{
			&TextMessage{"Hello, World!"},
			&ImageMessage{"http://example.com/image.jpg"},
		}
		// for _, msg := range messages {
		// countingWriter.BytesWritten = 0 // Reset counter

		// wrappedMsg := WrappedMessage{
		// 	Type:    msg.MsgType(),
		// 	Message: msg,
		// }

		if err := encoder.Encode(&messages); err != nil {
			fmt.Println("Error encoding message:", err)
			return
		}
		// fmt.Println("Sent message of type:", msg.MsgType())

		if err := zstdWriter.Flush(); err != nil {
			fmt.Println("Error flushing zlib writer:", err)
			return
		}

		// }
		fmt.Printf("Compressed data sent: %d bytes\n", buf.Len())

		time.Sleep(time.Second * 2)
	}
}
