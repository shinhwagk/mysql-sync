package main

import (
	"fmt"
	"time"
)

func main() {
	c1 := make(chan string, 100)

	go func() {
		time.Sleep(time.Second * 5)
		fmt.Println("send 2")

		for i := 0; i < 5; i++ {
			select {
			case c1 <- "asdfd":
				fmt.Println("send 2 send ")
			case <-time.After(time.Second * 1):
				fmt.Println("send 2 send timeout")
			}
		}

	}()

	go func() {
		time.Sleep(time.Second * 5)
		fmt.Println("send 3")

		for i := 0; i < 5; i++ {
			select {
			case c1 <- "asdfd":
				fmt.Println("send 3 send ")
			case <-time.After(time.Second * 1):
				fmt.Println("send 3 send timeout")
			}
		}
	}()
	go func() {
		time.Sleep(time.Second * 8)

	}()

	fmt.Println(len(c1))
	time.Sleep(time.Second * 51)

}
