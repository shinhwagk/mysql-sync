package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	var sm sync.Map
	var wg sync.WaitGroup

	// 协程：添加元素
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000000; i++ {
			if i%2 == 0 { // 每隔一个数字添加一个元素
				sm.Store(i, fmt.Sprintf("value%d", i))
				fmt.Printf("Added: %d\n", i)
				time.Sleep(500 * time.Millisecond) // 0.5秒添加一次，减慢添加速度
			}
		}
	}()

	// 协程：删除元素
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000000; i++ {
			if i%3 == 0 { // 每隔两个数字尝试删除一个元素
				sm.Delete(i)
				fmt.Printf("Deleted: %d\n", i)
				time.Sleep(300 * time.Millisecond) // 0.3秒删除一次，增加删除频率
			}
		}
	}()

	// 协程：遍历 map
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(1 * time.Second) // 每1秒遍历一次 map
		defer ticker.Stop()
		for range ticker.C {
			fmt.Println("Starting a new range iteration:")
			sm.Range(func(key, value interface{}) bool {
				fmt.Printf("Key: %v, Value: %v\n", key, value)
				return true
			})
		}
	}()

	// 等待所有协程完成
	wg.Wait()
	fmt.Println("All goroutines completed.")
}
