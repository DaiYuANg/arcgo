---
title: 'dix 快速开始'
linkTitle: 'getting-started'
description: '构建、启动与停止强类型模块图'
weight: 2
---

## Getting Started

本页给出一个可直接复制运行的 **`dix` 最小示例**：

- 定义若干强类型服务
- 组装为模块
- 直接从 `App` 调用 `Start()`
- 再通过 `Stop()` 管理生命周期

## 1) Install

```bash
go get github.com/DaiYuANg/arcgo/dix@latest
go get github.com/DaiYuANg/arcgo/logx@latest
```

## 2）创建 `main.go`

```go
package main

import (
	"context"
	"log/slog"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/DaiYuANg/arcgo/logx"
)

type Config struct {
	Port int
}

type Server struct {
	Logger *slog.Logger
	Config Config
}

func main() {
	configModule := dix.NewModule("config",
		dix.Providers(dix.Provider0(func() Config { return Config{Port: 8080} })),
	)

	serverModule := dix.NewModule("server",
		dix.Imports(configModule),
		dix.Providers(
			dix.Provider2(func(logger *slog.Logger, cfg Config) *Server {
				return &Server{Logger: logger, Config: cfg}
			}),
		),
		dix.Hooks(
			dix.OnStart(func(ctx context.Context, srv *Server) error {
				srv.Logger.Info("server starting", "port", srv.Config.Port)
				return nil
			}),
			dix.OnStop(func(ctx context.Context, srv *Server) error {
				srv.Logger.Info("server stopping", "port", srv.Config.Port)
				return nil
			}),
		),
	)

	logger, err := logx.NewDevelopment()
	if err != nil {
		panic(err)
	}

	app := dix.New(
		"demo",
		dix.UseProfile(dix.ProfileDev),
		dix.UseLogger(logger),
		dix.Modules(configModule, serverModule),
	)

	report := app.ValidateReport()
	if err := report.Err(); err != nil {
		panic(err)
	}
	for _, warning := range report.Warnings {
		logger.Warn("validation warning", "kind", warning.Kind, "module", warning.Module, "label", warning.Label)
	}

	rt, err := app.Start(context.Background())
	if err != nil {
		panic(err)
	}
	defer func() {
		_, _ = rt.StopWithReport(context.Background())
	}()
}
```

## 3）运行

```bash
go mod init example.com/dix-hello
go get github.com/DaiYuANg/arcgo/dix@latest
go get github.com/DaiYuANg/arcgo/logx@latest
go run .
```

## 可选：从 DI 解析框架 logger（结合 `logx`）

如果你希望 `dix` 框架内部日志直接使用模块图里构建的 logger，优先使用 `dix.UseLogger0/1/...`。

下面的示例使用了新的短模块 option 和 App option 写法；旧的 `WithModule*`、`With*` 以及 `WithLoggerFrom...` 形式仍然作为兼容入口保留。

```go
package main

import (
	"context"
	"log/slog"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/DaiYuANg/arcgo/logx"
)

type LogBundle struct {
	Logger *slog.Logger
}

func main() {
	logModule := dix.NewModule("logx",
		dix.Providers(
			dix.Provider0(func() *LogBundle {
				return &LogBundle{
					Logger: logx.MustNew(logx.WithConsole(true), logx.WithDebugLevel()),
				}
			}),
		),
		dix.Hooks(
			dix.OnStop(func(_ context.Context, logs *LogBundle) error {
				return logx.Close(logs.Logger)
			}),
		),
	)

	app := dix.New(
		"demo",
		dix.Modules(logModule /*, other modules... */),
		dix.UseLogger1(func(logs *LogBundle) *slog.Logger {
			return logs.Logger
		}),
	)

	_, _ = app.Build()
}
```

这样可以把 logger 的初始化和回收都放在模块里，同时覆盖掉框架默认 logger。

## 可选：完全接管 dix 内部事件日志

如果你希望完全控制 dix 内部 build/start/stop/health/debug 输出，可以使用 `dix.UseEventLogger...`。
它和 `Observer` 不同，前者是主日志入口，后者更适合作为旁路订阅。

```go
type MyEventLogger struct{}

func (l *MyEventLogger) LogEvent(ctx context.Context, event dix.Event) {
	_ = ctx
	_ = event
}

app := dix.New(
	"demo",
	dix.Modules(logModule),
	dix.UseEventLogger0(func() dix.EventLogger {
		return &MyEventLogger{}
	}),
)
```

## 校验说明

- 对纯 typed 应用来说，`app.Validate()` 通常已经足够。
- 一旦用了 raw bridge API，更推荐 `app.ValidateReport()`，这样既能看到硬错误，也能看到 warning。
- 如果 raw 路径是有意为之，优先使用带 metadata 的 API 显式声明校验边界，而不是完全不透明的 escape hatch。

## 可选：由调用方控制运行上下文

如果你的进程已经有统一管理的 context，优先使用 `app.RunContext(ctx)`，而不是 `app.Run()`：

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

if err := app.RunContext(ctx); err != nil {
	panic(err)
}
```

## Next

- 运行时指标、Prometheus、OTel：[指标与可观测性](./metrics-and-observability)
- 健康检查与 `net/http` handler：[健康检查与生命周期](./health-and-lifecycle)
- 高级特性（named/alias/scope/override）：见 [dix 示例](./examples) 与 `dix/advanced`
