package main

import (
	"fmt"
)

func main() {
	// 模拟从binlog接收到的数据
	var data []byte // 这里假设data是从binlog读出来的，现在是nil

	// 检查data是否为nil，并适当处理
	a := []byte(nil)
	b := []byte{}

	// 检查是否为 nil
	fmt.Println(a == nil, len(a))       // 输出: true
	fmt.Println(b == nil, len(b))       // 输出: false
	fmt.Println(data == nil, len(data)) // 输出: false

	// 现在data是一个空的byte切片，不再是nil

}
