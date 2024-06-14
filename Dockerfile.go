FROM golang:1.22.3 as builder

WORKDIR /build

COPY go.mod .
COPY go.sum .
COPY src/ .

RUN go mod download

RUN go build -ldflags="-s -w" -o replication replication.go logger.go metric.go operation.go config.go tcpserver.go extract.go

RUN go build -ldflags="-s -w" -o destination destination.go logger.go metric.go operation.go config.go tcpclient.go hjdb.go mysql.go applier.go

COPY entrypoint.sh . 
RUN chmod +x entrypoint.sh

FROM golang:1.22.3-alpine

WORKDIR /app

COPY --from=builder /build/replication .
COPY --from=builder /build/destination .
COPY --from=builder /build/entrypoint.sh .

ENTRYPOINT ["/app/entrypoint.sh"]

CMD ["--help"]
