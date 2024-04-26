#!/bin/bash
go build -ldflags="-w -s" -o main_go main.go


GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o main_go_amd64 main.go && upx -9 main_go_amd64

