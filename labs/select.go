package main

import (
	"context"
	"fmt"
	"time"
)

func Start(ctx context.Context, xxx chan string) {
	timer := time.NewTimer(100 * time.Millisecond)
	for {
		fmt.Println("xxxxx")
		select {

		case <-ctx.Done():
			return
		case x, ok := <-xxx:
			if !ok {
				return
			}
			fmt.Println(ok)
			fmt.Println(x)
			timer.Reset(2 * time.Second)

		case <-timer.C:
			timer.Reset(2 * time.Second)
			fmt.Println("No data received for 5 minutes, stopping.")
		}

	}

}
func main() {

	mdCtx, _ := context.WithCancel(context.Background())

	sch := make(chan string)

	go func() {
		Start(mdCtx, sch)
		fmt.Println("stop")
	}()

	go func() {
		for {
			sch <- "yyyyyy"
			time.Sleep(time.Second * 11)
		}
	}()

	time.Sleep(time.Second * 101)
	close(sch)
	fmt.Println("xxx1111xx")

	time.Sleep(time.Second * 1000)

}
