FROM golang:1.22.4 as builder

WORKDIR /build
COPY go.mod .
COPY go.sum .
COPY src/ .
COPY vendor vendor

RUN go mod tidy
RUN go mod vendor
RUN go build -ldflags="-s -w" -o mysqlsync ./*.go

FROM golang:1.22.4
WORKDIR /app
COPY --from=builder /build/mysqlsync .
ENTRYPOINT ["/app/mysqlsync"]
CMD ["--help"]
