package dix

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/DaiYuANg/archgo/collectionx"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// SupportedFxLoggerType 表示这个库当前支持用“类型参数”自动映射的 logger 类型。
// - *slog.Logger       -> fxevent.SlogLogger
// - *zap.Logger        -> fxevent.ZapLogger
// - *fxevent.ConsoleLogger -> 直接使用 ConsoleLogger（默认写 stderr）
type SupportedFxLoggerType interface {
	*slog.Logger | *zap.Logger | *fxevent.ConsoleLogger
}

// CreateApplicationContainer 会：
// 1. 根据泛型类型 L 自动附加对应的 Fx logger option
// 2. 拼上调用方传入的所有 fx.Option
// 3. 先 ValidateApp
// 4. 校验通过后再 fx.New
func CreateApplicationContainer[L SupportedFxLoggerType](modules ...fx.Option) (*fx.App, error) {
	opts := collectionx.NewList[fx.Option]()
	opts.Add(loggerOption[L]())
	opts.MergeSlice(modules)
	built := opts.Values()

	if err := fx.ValidateApp(built...); err != nil {
		return nil, fmt.Errorf("validate fx app failed: %w", err)
	}

	app := fx.New(built...)
	return app, nil
}

// loggerOption 根据类型参数自动生成对应的 fx.WithLogger(...)。
func loggerOption[L SupportedFxLoggerType]() fx.Option {
	var zero L

	switch any(zero).(type) {
	case *slog.Logger:
		return fx.WithLogger(func(logger *slog.Logger) fxevent.Logger {
			return &fxevent.SlogLogger{Logger: logger}
		})

	case *zap.Logger:
		return fx.WithLogger(func(logger *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: logger}
		})

	case *fxevent.ConsoleLogger:
		// 这里不依赖容器里的 ConsoleLogger，直接构造一个默认 console logger。
		// Fx 自己默认 fallback 也是 ConsoleLogger 写 stderr。
		return fx.WithLogger(func() fxevent.Logger {
			return &fxevent.ConsoleLogger{W: os.Stderr}
		})

	default:
		// 理论上约束已经兜住了，这里只是防御式分支。
		panic(fmt.Sprintf("unsupported fx logger type: %T", zero))
	}
}
