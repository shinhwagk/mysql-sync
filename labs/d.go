package main

import (
	"fmt"
	"math"
	"time"
)

// SendResult 代表一次发送操作的结果
type SendResult struct {
	SendQuantity int
	Delay        float64
	Throughput   float64
	Timestamp    time.Time
}

// AdjustSendQuantity 根据当前的发送数据和历史数据，调整发送数量
func AdjustSendQuantity(
	currentDelay float64,
	currentSendQuantity int,
	history []SendResult,
) (int, []SendResult) {
	// 定义默认的阈值参数（可以根据需要调整）
	const (
		deviationThreshold = 2.0 // 延迟偏离平均值的标准差倍数阈值
	)

	sendQuantity := currentSendQuantity

	// 计算当前的吞吐量
	throughput := float64(currentSendQuantity) / currentDelay

	// 保存当前数据到历史记录
	history = append(history, SendResult{
		SendQuantity: currentSendQuantity,
		Delay:        currentDelay,
		Throughput:   throughput,
		Timestamp:    time.Now(),
	})

	// 计算历史延迟的平均值和标准差
	meanDelay, stdDevDelay := calculateMeanAndStdDev(history)

	// 检查当前延迟是否偏离历史平均值超过阈值
	if len(history) > 1 && stdDevDelay > 0 {
		if math.Abs(currentDelay-meanDelay) > deviationThreshold*stdDevDelay {
			fmt.Printf("检测到延迟波动: 当前延迟偏离平均值 %.2f 个标准差\n", math.Abs(currentDelay-meanDelay)/stdDevDelay)
			// 减少发送数量
			sendQuantity = int(float64(sendQuantity) * 0.9) // 减少发送数量10%
			if sendQuantity < 1 {
				sendQuantity = 1
			}
			fmt.Printf("由于波动，减少发送数量。新的发送数量: %d\n", sendQuantity)
			return sendQuantity, history
		}
	}

	// 检查吞吐量是否有改进
	prevThroughput := 0.0
	if len(history) > 1 {
		prevThroughput = history[len(history)-2].Throughput
	}
	if throughput > prevThroughput {
		// 吞吐量有改进，增加发送数量
		sendQuantity += 1
		fmt.Printf("吞吐量提高，增加发送数量。新的发送数量: %d\n", sendQuantity)
	} else {
		// 吞吐量没有改进，保持发送数量
	}

	// 根据历史数据调整发送数量
	sendQuantity = adjustBasedOnHistory(history, sendQuantity)

	return sendQuantity, history
}

// calculateMeanAndStdDev 计算历史延迟的平均值和标准差
func calculateMeanAndStdDev(history []SendResult) (mean, stdDev float64) {
	if len(history) == 0 {
		return 0, 0
	}

	sum := 0.0
	for _, result := range history {
		sum += result.Delay
	}
	mean = sum / float64(len(history))

	varianceSum := 0.0
	for _, result := range history {
		varianceSum += (result.Delay - mean) * (result.Delay - mean)
	}
	stdDev = math.Sqrt(varianceSum / float64(len(history)))

	return mean, stdDev
}

// adjustBasedOnHistory 根据历史数据进一步调整发送数量
func adjustBasedOnHistory(history []SendResult, currentQuantity int) int {
	if len(history) < 3 {
		return currentQuantity
	}

	// 检查最近三次的吞吐量是否持续下降
	last := history[len(history)-1]
	secondLast := history[len(history)-2]
	thirdLast := history[len(history)-3]

	if last.Throughput < secondLast.Throughput && secondLast.Throughput < thirdLast.Throughput {
		// 减少发送数量
		newQuantity := currentQuantity - 1
		if newQuantity < 1 {
			newQuantity = 1
		}
		fmt.Printf("吞吐量连续下降，调整发送数量为 %d\n", newQuantity)
		return newQuantity
	}

	return currentQuantity
}

// 以下是一个使用示例
func main() {
	history := []SendResult{}
	sendQuantity := 1

	// 模拟运行多次调整
	for i := 0; i < 20; i++ {
		// 假设您已经测量了当前发送的延迟
		currentDelay := simulateDelay(sendQuantity)
		fmt.Printf("第 %d 次发送，发送数量: %d, 延迟: %.4f\n", i+1, sendQuantity, currentDelay)

		// 调用 AdjustSendQuantity 函数调整发送数量
		sendQuantity, history = AdjustSendQuantity(
			currentDelay,
			sendQuantity,
			history,
		)

		// 您可以在这里添加实际的发送逻辑
	}
}

// simulateDelay 模拟发送延迟（可根据实际情况替换为真实的延迟测量）
func simulateDelay(quantity int) float64 {
	baseDelay := 1.0
	variance := 0.2

	// 使用正弦函数模拟波动
	timeFactor := float64(time.Now().UnixNano()%1e9) / 1e9 * 2 * math.Pi
	fluctuation := math.Sin(timeFactor*5) * 0.1

	delay := baseDelay + (math.Log(float64(quantity+1)) * 0.5)
	delay += (variance * (randFloat() - 0.5))
	delay += fluctuation

	if delay < 0.1 {
		delay = 0.1
	}

	return delay
}

// randFloat 返回 0 到 1 之间的随机浮点数
func randFloat() float64 {
	return float64(time.Now().UnixNano()%1e3) / 1e3
}
