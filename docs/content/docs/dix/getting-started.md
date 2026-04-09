---
title: 'dix Getting Started'
linkTitle: 'getting-started'
description: 'Build, start, and stop a typed module graph'
weight: 2
---

## Getting Started

This page shows a **self-contained** `dix` program:

- define a couple of typed services
- compose them into modules
- `Start()` a runtime directly from `App`
- `Stop()` it cleanly

## 1) Install

```bash
go get github.com/DaiYuANg/arcgo/dix@latest
go get github.com/DaiYuANg/arcgo/logx@latest
```

## 2) Create `main.go`

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

## 3) Run

```bash
go mod init example.com/dix-hello
go get github.com/DaiYuANg/arcgo/dix@latest
go get github.com/DaiYuANg/arcgo/logx@latest
go run .
```

## Optional: resolve framework logger from DI (`logx`)

If you want `dix` internal logs to use a logger produced by your module graph, prefer `dix.UseLogger0/1/...`.

The examples below use the newer short module and app option aliases. The older `WithModule*`, `With*`, and `WithLoggerFrom...` forms remain valid as compatibility entry points.

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

This keeps logger wiring inside modules while still replacing the framework default logger.

## Optional: fully own dix internal event logging

If you want full control over dix internal build/start/stop/health/debug output, use `dix.UseEventLogger...`.
Unlike `Observer`, this replaces the primary dix logging path.

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

## Validation notes

- For typed-only apps, `app.Validate()` is usually enough.
- When you use raw bridge APIs, prefer `app.ValidateReport()` so you can inspect warnings as well as hard errors.
- If a raw path is intentional, declare its validation boundary with metadata-aware APIs instead of relying on a fully opaque escape hatch.

## Optional: run with caller-owned context

If your process already has a managed context, prefer `app.RunContext(ctx)` over `app.Run()`:

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

if err := app.RunContext(ctx); err != nil {
	panic(err)
}
```

## Next

- Runtime metrics, Prometheus, and OTel: [Metrics and observability](./metrics-and-observability)
- Health checks and `net/http` handlers: [Health and lifecycle](./health-and-lifecycle)
- Advanced features (named/alias/scope/override): see [dix examples](./examples) and `dix/advanced`
