FROM golang:1.22.4 as builder

WORKDIR /build
COPY go.mod .
COPY go.sum .
COPY src/ .
COPY vendor vendor

RUN go mod tidy
RUN go mod vendor
# RUN go build -ldflags="-s -w" -o mysqlsync ./*.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-s -w" -o mysqlsync ./*.go

FROM scratch
WORKDIR /app
COPY --from=builder /build/mysqlsync .
ENTRYPOINT ["/app/mysqlsync"]
CMD ["--help"]

# FROM alpine:3.20
# WORKDIR /app
# COPY --from=builder /build/mysqlsync /app/mysqlsync
# # RUN apk add --no-cache ca-certificates
# ENTRYPOINT ["/app/mysqlsync"]
# CMD ["--help"]
