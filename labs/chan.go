package main

import (
	"fmt"
	"time"
)

func main() {
	c1 := make(chan string, 100)

	go func() {
		time.Sleep(time.Second)

		fmt.Println(len(c1))

		c1 <- "asdfd"
		fmt.Println(len(c1))

		c1 <- "asdfd"
		fmt.Println(len(c1))

		close(c1) // 发送完毕后关闭通道
	}()

	close(c1)
	fmt.Println(len(c1))

	<-c1
	fmt.Println(len(c1))
	time.Sleep(time.Second * 5)

}
