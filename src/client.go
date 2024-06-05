package main

// import (
// 	"bytes"
// 	"compress/gzip"
// 	"encoding/binary"
// 	"encoding/gob"
// 	"io"
// 	"log"
// 	"net"
// )

// type MyObject struct {
// 	SerialNumber int
// 	Name         string
// 	Value        int
// }

// func Client() {
// 	conn, err := net.Dial("tcp", "localhost:9999")
// 	if err != nil {
// 		log.Fatalf("Error connecting to server: %v", err)
// 	}
// 	defer conn.Close()

// 	// 发送起始序列号
// 	startSerial := 20
// 	err = binary.Write(conn, binary.BigEndian, startSerial)
// 	if err != nil {
// 		log.Fatalf("Error writing serial number: %v", err)
// 	}

// 	// 不断接收和处理来自服务器的数据
// 	for {
// 		var compressedData bytes.Buffer
// 		_, err := io.Copy(&compressedData, conn)
// 		if err != nil {
// 			log.Fatalf("Error reading data: %v", err)
// 			break
// 		}

// 		gzReader, err := gzip.NewReader(&compressedData)
// 		if err != nil {s
// 			log.Fatalf("Failed to create gzip reader: %v", err)
// 		}
// 		uncompressedData, err := io.ReadAll(gzReader)
// 		gzReader.Close()
// 		if err != nil {
// 			log.Fatalf("Failed to read uncompressed data: %v", err)
// 			continue
// 		}

// 		var obj MyObject
// 		dec := gob.NewDecoder(bytes.NewReader(uncompressedData))
// 		if err := dec.Decode(&obj); err != nil {
// 			log.Fatalf("Failed to decode: %v", err)
// 			continue
// 		}

// 		log.Printf("Received Object: %+v", obj)
// 	}
// }
