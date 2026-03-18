package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/DaiYuANg/arcgo/eventx"
	eventxfx "github.com/DaiYuANg/arcgo/eventx/fx"
	"github.com/DaiYuANg/arcgo/logx"
	logxfx "github.com/DaiYuANg/arcgo/logx/fx"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

// NotificationEvent 通知事件
type NotificationEvent struct {
	Type    string
	UserID  string
	Message string
}

func (e NotificationEvent) Name() string {
	return "notification." + e.Type
}

func main() {
	fmt.Println("=== EventX + FX + LogX 集成示例 ===")

	app := fx.New(
		fx.WithLogger(func(log *slog.Logger) fxevent.Logger {
			return &fxevent.SlogLogger{Logger: log}
		}),
		// 使用 logx fxx 模块（带 slog 支持）
		logxfx.NewLogxModuleWithSlog(
			logx.WithLevel(logx.DebugLevel),
			logx.WithCaller(true),
		),

		// 使用 eventx fxx 模块
		eventxfx.NewEventxModule(
			eventx.WithAntsPool(4),
			eventx.WithParallelDispatch(true),
		),

		// 初始化订阅者
		fx.Invoke(func(bus eventx.BusRuntime, logger *slog.Logger) {
			logger.Info("🚀 正在注册通知订阅者...")

			// 订阅邮件通知
			_, err := eventx.Subscribe[NotificationEvent](bus,
				func(ctx context.Context, event NotificationEvent) error {
					if event.Type != "email" {
						return nil
					}
					logx.WithFields(logger, map[string]any{
						"user_id":  event.UserID,
						"msg_type": "email",
					}).Info("📧 发送邮件")
					fmt.Printf("   📧 发送邮件给用户 %s: %s\n", event.UserID, event.Message)
					time.Sleep(100 * time.Millisecond)
					return nil
				},
			)
			if err != nil {
				logx.WithError(logger, err).Error("注册邮件订阅者失败")
				panic(err)
			}

			// 订阅短信通知
			_, err = eventx.Subscribe[NotificationEvent](bus,
				func(ctx context.Context, event NotificationEvent) error {
					if event.Type != "sms" {
						return nil
					}
					logx.WithFields(logger, map[string]any{
						"user_id":  event.UserID,
						"msg_type": "sms",
					}).Info("📱 发送短信")
					fmt.Printf("   📱 发送短信给用户 %s: %s\n", event.UserID, event.Message)
					time.Sleep(100 * time.Millisecond)
					return nil
				},
			)
			if err != nil {
				logx.WithError(logger, err).Error("注册短信订阅者失败")
				panic(err)
			}

			// 订阅推送通知
			_, err = eventx.Subscribe[NotificationEvent](bus,
				func(ctx context.Context, event NotificationEvent) error {
					if event.Type != "push" {
						return nil
					}
					logx.WithFields(logger, map[string]any{
						"user_id":  event.UserID,
						"msg_type": "push",
					}).Info("🔔 发送推送")
					fmt.Printf("   🔔 发送推送给用户 %s: %s\n", event.UserID, event.Message)
					time.Sleep(100 * time.Millisecond)
					return nil
				},
			)
			if err != nil {
				logx.WithError(logger, err).Error("注册推送订阅者失败")
				panic(err)
			}

			logger.Info("✅ 所有通知订阅者已注册完成")
		}),

		// 运行业务逻辑
		fx.Invoke(func(lc fx.Lifecycle, bus eventx.BusRuntime, logger *slog.Logger) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					logger.Info("📨 开始发布通知事件...")
					fmt.Println("\n=== 发布通知事件 ===")

					// 发布多个通知
					events := []NotificationEvent{
						{Type: "email", UserID: "user-001", Message: "欢迎注册！"},
						{Type: "sms", UserID: "user-001", Message: "验证码：123456"},
						{Type: "push", UserID: "user-002", Message: "您有新的消息"},
						{Type: "email", UserID: "user-002", Message: "订单已发货"},
					}

					for i, event := range events {
						logx.WithFields(logger, map[string]any{
							"index":   i + 1,
							"type":    event.Type,
							"user_id": event.UserID,
							"total":   len(events),
						}).Info("发布通知事件")

						err := bus.PublishAsync(context.Background(), event)
						if err != nil {
							logx.WithError(logx.WithFields(logger, map[string]any{
								"event": event,
							}), err).Error("发布事件失败")
						}
					}

					logger.Info("✅ 所有通知已发布到异步队列")
					fmt.Println("\n✅ 所有通知已发布到异步队列")

					// 等待异步事件处理完成
					time.Sleep(500 * time.Millisecond)
					return nil
				},
			})
		}),
	)

	if err := app.Start(context.Background()); err != nil {
		panic(err)
	}
}
