---
title: 'dix 健康检查与生命周期'
linkTitle: 'health-lifecycle'
description: '注册健康检查并暴露健康检查端点'
weight: 3
---

## 健康检查与生命周期

`dix` 运行时支持三类检查：

- 通用健康检查（`CheckHealth`）
- 存活检查（`CheckLiveness`）
- 就绪检查（`CheckReadiness`）

通常在 `Setups(...)`、`SetupContainer(...)` 或 `WithModuleSetup(...)` 中通过 `*dix.Container` 注册检查。对 HTTP 场景，`Runtime` 也提供可直接挂载的 handler：

- `rt.HealthHandler()` → `/healthz`
- `rt.LivenessHandler()` → `/livez`
- `rt.ReadinessHandler()` → `/readyz`

## 1）安装

```bash
go get github.com/DaiYuANg/arcgo/dix@latest
go get github.com/DaiYuANg/arcgo/logx@latest
```

## 2）创建 `main.go`

本示例注册一个始终通过的 liveness 检查，并注册一个依赖 `*Server` 可解析的 readiness 检查。

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

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

	logger, err := logx.NewDevelopment()
	if err != nil {
		panic(err)
	}

	serverModule := dix.NewModule("server",
		dix.Imports(configModule),
		dix.Providers(
			dix.Provider2(func(logger *slog.Logger, cfg Config) *Server {
				return &Server{Logger: logger, Config: cfg}
			}),
		),
		dix.Setups(dix.SetupContainer(func(c *dix.Container) error {
			c.RegisterLivenessCheck("process", func(context.Context) error { return nil })
			c.RegisterReadinessCheck("bootstrap", func(context.Context) error {
				server, ok := dix.ResolveOptionalAs[*Server](c)
				if !ok || server == nil {
					return errors.New("server not ready")
				}
				return nil
			})
			return nil
		})),
	)

	app := dix.NewDefault(
		dix.UseProfile(dix.ProfileDev),
		dix.Version("0.1.0"),
		dix.Modules(serverModule),
		dix.UseLogger(logger),
	)

	rt, err := app.Start(context.Background())
	if err != nil {
		panic(err)
	}
	defer func() { _ = rt.Stop(context.Background()) }()

	fmt.Println("health:", rt.CheckHealth(context.Background()).Healthy())
	fmt.Println("liveness:", rt.CheckLiveness(context.Background()).Healthy())
	fmt.Println("readiness:", rt.CheckReadiness(context.Background()).Healthy())

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", rt.HealthHandler())
	mux.HandleFunc("/livez", rt.LivenessHandler())
	mux.HandleFunc("/readyz", rt.ReadinessHandler())

_ = mux
}
```

## 可选：使用 DI logger 作为框架 logger（`logx`）

如果你的健康检查/生命周期模块也负责 logger 初始化，可以让 `dix` 框架内部日志直接使用 DI 构建出的 logger：

```go
type LogBundle struct {
	Logger *slog.Logger
}

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

app := dix.NewDefault(
	dix.Modules(logModule, serverModule),
	dix.UseLogger1(func(logs *LogBundle) *slog.Logger {
		return logs.Logger
	}),
)
```

这样可以把 logger 的生命周期放在模块里统一管理，同时替换掉框架默认 logger。

## 延伸阅读

- [快速开始](./getting-started)
- 示例导航：[dix examples](./examples)
