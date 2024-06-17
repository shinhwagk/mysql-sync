FROM golang:1.22.3 as builder

WORKDIR /build
COPY go.mod .
COPY go.sum .
COPY src/ .

RUN go mod download
RUN go build -ldflags="-s -w" -o mysqlsync ./*.go

FROM golang:1.22.3-alpine
WORKDIR /app
COPY --from=builder /build/mysqlsync .
ENTRYPOINT ["/app/mysqlsync"]
CMD ["--help"]
