package main

import (
	"context"
	"fmt"
	"time"

	"github.com/DaiYuANg/archgo/eventx"
)

// UserEvent 用户事件基类
type UserEvent struct {
	EventName string
	UserID    string
	Timestamp time.Time
}

func (e UserEvent) Name() string {
	return e.EventName
}

// UserRegisteredEvent 用户注册事件
type UserRegisteredEvent struct {
	UserEvent
	Email    string
	UserName string
}

func (e UserRegisteredEvent) Name() string {
	return e.UserEvent.Name()
}

// UserLoginEvent 用户登录事件
type UserLoginEvent struct {
	UserEvent
	IPAddress string
}

func (e UserLoginEvent) Name() string {
	return e.UserEvent.Name()
}

func main() {
	fmt.Println("=== EventX 中间件示例 ===")

	// 创建带中间件的事件总线
	bus := eventx.New(
		eventx.WithAntsPool(4),
		// 全局中间件：日志记录
		eventx.WithMiddleware(func(next eventx.HandlerFunc) eventx.HandlerFunc {
			return func(ctx context.Context, event eventx.Event) error {
				start := time.Now()
				fmt.Printf("🔍 [中间件] 开始处理事件：%s\n", event.Name())
				err := next(ctx, event)
				duration := time.Since(start)
				fmt.Printf("✅ [中间件] 事件处理完成：%s, 耗时：%v\n", event.Name(), duration)
				return err
			}
		}),
		// 全局中间件：恢复 panic
		eventx.WithMiddleware(eventx.RecoverMiddleware()),
	)
	defer func() {
		_ = bus.Close()
	}()

	// 订阅用户注册事件 - 带订阅者中间件
	_, err := eventx.Subscribe[UserRegisteredEvent](bus,
		func(ctx context.Context, event UserRegisteredEvent) error {
			fmt.Printf("  👤 处理用户注册：%s (%s)\n", event.UserName, event.Email)

			// 模拟可能 panic 的操作
			if event.Email == "panic@example.com" {
				panic("模拟处理异常！")
			}

			time.Sleep(100 * time.Millisecond)
			return nil
		},
		// 订阅者级别的中间件：权限检查
		eventx.WithSubscriberMiddleware(func(next eventx.HandlerFunc) eventx.HandlerFunc {
			return func(ctx context.Context, event eventx.Event) error {
				fmt.Printf("  🔐 [权限检查] 验证事件权限\n")
				return next(ctx, event)
			}
		}),
	)
	if err != nil {
		panic(err)
	}

	// 订阅用户登录事件
	_, err = eventx.Subscribe[UserLoginEvent](bus, func(ctx context.Context, event UserLoginEvent) error {
		fmt.Printf("  🔑 处理用户登录：%s from %s\n", event.UserID, event.IPAddress)
		time.Sleep(50 * time.Millisecond)
		return nil
	})
	if err != nil {
		panic(err)
	}

	// 发布正常事件
	fmt.Println("--- 发布用户注册事件（正常） ---")
	err = bus.Publish(context.Background(), UserRegisteredEvent{
		UserEvent: UserEvent{
			EventName: "user.registered",
			UserID:    "user-001",
			Timestamp: time.Now(),
		},
		Email:    "john@example.com",
		UserName: "John Doe",
	})
	if err != nil {
		fmt.Printf("错误：%v\n", err)
	}

	fmt.Println("\n--- 发布用户登录事件 ---")
	err = bus.Publish(context.Background(), UserLoginEvent{
		UserEvent: UserEvent{
			EventName: "user.loggedin",
			UserID:    "user-001",
			Timestamp: time.Now(),
		},
		IPAddress: "192.168.1.100",
	})
	if err != nil {
		fmt.Printf("错误：%v\n", err)
	}

	fmt.Println("\n--- 发布用户注册事件（会 panic） ---")
	err = bus.Publish(context.Background(), UserRegisteredEvent{
		UserEvent: UserEvent{
			EventName: "user.registered",
			UserID:    "user-002",
			Timestamp: time.Now(),
		},
		Email:    "panic@example.com",
		UserName: "Panic User",
	})
	if err != nil {
		fmt.Printf("错误：%v\n", err)
	}

	fmt.Println("\n✅ 总线继续正常工作，panic 已被恢复中间件捕获")
}
