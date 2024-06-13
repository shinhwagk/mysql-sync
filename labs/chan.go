package main

import "fmt"

func main() {
	c1 := make(chan string, 100)

	go func() {
		fmt.Println(len(c1))

		c1 <- "asdfd"
		fmt.Println(len(c1))

		c1 <- "asdfd"
		fmt.Println(len(c1))

		close(c1) // 发送完毕后关闭通道
	}()

	fmt.Println(len(c1))

	<-c1
	fmt.Println(len(c1))

}
