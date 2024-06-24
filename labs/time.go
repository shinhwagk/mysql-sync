package main

import (
	"fmt"
	"time"
)

func main() {
	tickerFast := time.NewTicker(100 * time.Millisecond)
	defer tickerFast.Stop()

	tickerSlow := time.NewTicker(1 * time.Second)
	defer tickerSlow.Stop()

	for {
		select {
		case <-tickerFast.C:
			fmt.Println("Fast ticker ticked")
		case <-tickerSlow.C:
			fmt.Println("Slow ticker ticked")
		case <-time.After(3 * time.Second):
			fmt.Println("Timeout, exiting...")
			return
		}
	}
}
