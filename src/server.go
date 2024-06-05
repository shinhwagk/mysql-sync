package main

// import (
// 	"bytes"
// 	"compress/gzip"
// 	"encoding/binary"
// 	"encoding/gob"
// 	"log"
// 	"net"
// )

// type MyObject struct {
// 	SerialNumber int
// 	Name         string
// 	Value        int
// }

// var objects = []MyObject{
// 	{SerialNumber: 1, Name: "Object1", Value: 10},
// 	{SerialNumber: 2, Name: "Object2", Value: 20},

// 	{SerialNumber: 100, Name: "Object100", Value: 1000},
// }

// func Server() {
// 	listener, err := net.Listen("tcp", "localhost:9999")
// 	if err != nil {
// 		log.Fatal("Error listening:", err)
// 	}
// 	defer listener.Close()
// 	log.Println("Server listening on localhost:9999")

// 	for {
// 		conn, err := listener.Accept()
// 		if err != nil {
// 			log.Println("Error accepting:", err)
// 			continue
// 		}
// 		go handleConnection(conn)
// 	}
// }

// func handleConnection(conn net.Conn) {
// 	defer conn.Close()

// 	// 读取客户端发送的起始序列号
// 	var startSerial int
// 	err := binary.Read(conn, binary.BigEndian, &startSerial)
// 	if err != nil {
// 		log.Printf("Error reading serial number: %v", err)
// 		return
// 	}

// 	// 从指定序列号开始发送对象
// 	for _, obj := range objects {
// 		if obj.SerialNumber >= startSerial {
// 			if err := sendObject(conn, obj); err != nil {
// 				log.Printf("Error sending object: %v", err)
// 				return
// 			}
// 		}
// 	}
// }

// func sendObject(conn net.Conn, obj MyObject) error {
// 	var buf bytes.Buffer
// 	enc := gob.NewEncoder(&buf)
// 	if err := enc.Encode(obj); err != nil {
// 		return err
// 	}

// 	w := gzip.NewWriter(&buf)
// 	if _, err := w.Write(buf.Bytes()); err != nil {
// 		return err
// 	}
// 	w.Close()

// 	_, err := conn.Write(buf.Bytes())
// 	return err
// }
