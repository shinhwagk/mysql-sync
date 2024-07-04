package main

import (
	"fmt"
	"net/url"
)

func main() {
	baseURL := "your_database_connection_prefix_here" // 例如 "user:pass@tcp(host:port)/dbname"
	params := url.Values{}
	params.Add("autocommit", "false")
	params.Add("time_zone", "+00:00")

	// 构造完整的连接字符串
	connectionString := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	fmt.Println("Encoded URL:", connectionString)
}
