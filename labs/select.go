package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	moCh := make(chan int)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		for {
			fmt.Println("assssssss")
			select {
			case oper, ok := <-moCh:
				if !ok {
					fmt.Println("moCh closed")
					return
				}
				fmt.Println("Received from moCh:", oper)
			case <-ctx.Done():
				fmt.Println("Context cancelled")
				return
			}
			fmt.Println("xxxxxxxxx")
		}
	}()

	// Simulate sending data to moCh
	go func() {
		time.Sleep(11 * time.Second)
		moCh <- 1
		time.Sleep(2 * time.Second)
		moCh <- 2

		// Optionally cancel the context to see context cancellation handling
		// cancel()

		// Close the channel after use
		time.Sleep(3 * time.Second)
		close(moCh)
	}()

	// Wait to see the output
	time.Sleep(10 * time.Second)
	cancel() // Clean up the context
}
