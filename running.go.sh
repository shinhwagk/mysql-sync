#!/bin/bash

# replication
go run replication.go operation.go config.go logger.go metric.go tcpserver.go hjdb.go extract.go

# destination
go run destination.go logger.go hjdb.go mysql.go config.go operation.go tcpclient.go applier.go metric.go
