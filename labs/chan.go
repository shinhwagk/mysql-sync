package main

import (
	"fmt"
	"time"
)

func main() {
	c1 := make(chan int)
	go func() {
		for i := 0; i <= 2; i++ {
			fmt.Println("ru i")
			c1 <- i
			fmt.Println("ru i ok")

		}
	}()

	go func() {
		time.Sleep(time.Second * 2)
		for {
			select {
			case _, ok := <-c1:
				if !ok {
					fmt.Println("end")
					return
				}
				fmt.Println("Xxxx")
			case <-time.After(time.Second):
				fmt.Println("xxxx1")
				close(c1)
				fmt.Println("xxxx")
			}
		}
	}()

	time.Sleep(time.Second * 115)

}
