package main

import (
	"context"
	"fmt"
	"time"

	"github.com/DaiYuANg/archgo/eventx"
)

// OrderCreatedEvent 表示订单创建事件
type OrderCreatedEvent struct {
	OrderID   string
	UserID    string
	Amount    float64
	CreatedAt time.Time
}

func (e OrderCreatedEvent) Name() string {
	return "order.created"
}

// OrderPaidEvent 表示订单支付事件
type OrderPaidEvent struct {
	OrderID string
	PaidAt  time.Time
}

func (e OrderPaidEvent) Name() string {
	return "order.paid"
}

func main() {
	// 创建事件总线
	bus := eventx.New(
		eventx.WithAntsPool(4), // 使用 4 个协程的协程池
		eventx.WithParallelDispatch(true),
	)
	defer func() {
		_ = bus.Close()
	}()

	// 订阅订单创建事件 - 发送欢迎邮件
	_, err := eventx.Subscribe[OrderCreatedEvent](bus, func(ctx context.Context, event OrderCreatedEvent) error {
		fmt.Printf("📧 发送欢迎邮件给用户 %s (订单：%s)\n", event.UserID, event.OrderID)
		time.Sleep(100 * time.Millisecond) // 模拟发送邮件
		return nil
	})
	if err != nil {
		panic(err)
	}

	// 订阅订单创建事件 - 初始化订单数据
	_, err = eventx.Subscribe[OrderCreatedEvent](bus, func(ctx context.Context, event OrderCreatedEvent) error {
		fmt.Printf("📊 初始化订单数据：%s, 金额：%.2f\n", event.OrderID, event.Amount)
		return nil
	})
	if err != nil {
		panic(err)
	}

	// 订阅订单支付事件 - 更新库存
	_, err = eventx.Subscribe[OrderPaidEvent](bus, func(ctx context.Context, event OrderPaidEvent) error {
		fmt.Printf("📦 更新库存，订单：%s\n", event.OrderID)
		return nil
	})
	if err != nil {
		panic(err)
	}

	// 订阅订单支付事件 - 发送支付成功通知
	_, err = eventx.Subscribe[OrderPaidEvent](bus, func(ctx context.Context, event OrderPaidEvent) error {
		fmt.Printf("📱 发送支付成功通知，订单：%s\n", event.OrderID)
		return nil
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("=== 发布订单创建事件（同步） ===")
	err = bus.Publish(context.Background(), OrderCreatedEvent{
		OrderID:   "ORD-001",
		UserID:    "USER-123",
		Amount:    299.99,
		CreatedAt: time.Now(),
	})
	if err != nil {
		fmt.Printf("发布事件失败：%v\n", err)
	}

	fmt.Println("\n=== 发布订单支付事件（异步） ===")
	err = bus.PublishAsync(context.Background(), OrderPaidEvent{
		OrderID: "ORD-001",
		PaidAt:  time.Now(),
	})
	if err != nil {
		fmt.Printf("发布异步事件失败：%v\n", err)
	}

	// 等待异步事件处理完成
	time.Sleep(500 * time.Millisecond)

	fmt.Println("\n=== 当前订阅者数量:", bus.SubscriberCount(), "===")
}
