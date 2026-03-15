package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/DaiYuANg/archgo/eventx"
)

// StockEvent 库存事件
type StockEvent struct {
	ProductID string
	Change    int
	Reason    string
}

func (e StockEvent) Name() string {
	return "stock.change"
}

func main() {
	fmt.Println("=== EventX 高性能示例 - 批量处理 ===")

	var (
		totalProcessed int64
		totalErrors    int64
	)

	// 创建高性能事件总线
	bus := eventx.New(
		eventx.WithAntsPool(10),           // 10 个协程处理异步事件
		eventx.WithParallelDispatch(true), // 并行分发
		eventx.WithMiddleware(eventx.RecoverMiddleware()),
		eventx.WithAsyncErrorHandler(func(ctx context.Context, event eventx.Event, err error) {
			atomic.AddInt64(&totalErrors, 1)
			fmt.Printf("❌ 异步处理错误 [%s]: %v\n", event.Name(), err)
		}),
	)
	defer func() {
		_ = bus.Close()
	}()

	// 订阅库存变更事件 - 模拟多个消费者
	for i := 0; i < 3; i++ {
		consumerID := i
		_, err := eventx.Subscribe[StockEvent](bus, func(ctx context.Context, event StockEvent) error {
			atomic.AddInt64(&totalProcessed, 1)

			// 模拟处理逻辑
			time.Sleep(10 * time.Millisecond)

			// 模拟少量错误
			if event.Change < -100 {
				return fmt.Errorf("库存变更过大：%d", event.Change)
			}

			if consumerID == 0 {
				fmt.Printf("📦 消费者%d: 产品 %s 库存变更 %+d (%s)\n",
					consumerID, event.ProductID, event.Change, event.Reason)
			}
			return nil
		})
		if err != nil {
			panic(err)
		}
	}

	fmt.Println("开始批量发布库存变更事件...")
	startTime := time.Now()

	// 批量发布事件
	batchSize := 100
	for i := 0; i < batchSize; i++ {
		event := StockEvent{
			ProductID: fmt.Sprintf("PROD-%03d", i%10),
			Change:    -1,
			Reason:    "订单扣减",
		}

		err := bus.PublishAsync(context.Background(), event)
		if err != nil {
			fmt.Printf("发布失败：%v\n", err)
		}
	}

	// 等待处理完成
	time.Sleep(2 * time.Second)

	elapsed := time.Since(startTime)
	fmt.Printf("\n=== 处理完成 ===\n")
	fmt.Printf("⏱️  总耗时：%v\n", elapsed)
	fmt.Printf("📊 处理事件数：%d\n", atomic.LoadInt64(&totalProcessed))
	fmt.Printf("❌ 错误数：%d\n", atomic.LoadInt64(&totalErrors))
	fmt.Printf("🚀 吞吐量：%.0f 事件/秒\n", float64(batchSize)/elapsed.Seconds())
	fmt.Printf("📈 订阅者数量：%d\n", bus.SubscriberCount())
}
