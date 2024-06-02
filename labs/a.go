package main

import (
	"fmt"
	"time"
)

// 生产者函数，监听停止信号
func producer(ch chan<- int, stopChan <-chan struct{}) {
	i := 0
	for {
		select {
		case <-stopChan: // 接收到停止信号
			fmt.Println("Producer stopping")
			close(ch) // 关闭数据通道
			return
		default:
			ch <- i // 正常生产数据并发送到通道
			fmt.Println("Produced:", i)
			time.Sleep(time.Second) // 模拟耗时的生产过程
			i++
		}
	}
}

// 消费者函数，从通道接收数据并处理
func consumer(id int, ch <-chan int) {
	for value := range ch {
		fmt.Printf("Consumer %d got: %d\n", id, value)
		time.Sleep(time.Second * 2) // 模拟处理数据的时间
	}
	fmt.Printf("Consumer %d stopping\n", id)
}

func main() {
	ch := make(chan int)
	stopChan := make(chan struct{}) // 创建用于发送停止信号的通道

	// 启动生产者goroutine
	go producer(ch, stopChan)

	// 启动两个消费者goroutines
	go consumer(1, ch)
	go consumer(2, ch)

	// 模拟运行一段时间后发送停止信号
	time.Sleep(time.Second * 10)
	close(stopChan) // 发送停止信号给生产者

	// 等待一段时间让消费者处理完未完成的数据
	time.Sleep(time.Second * 10)
	fmt.Println("Main function ending")
}
