---
title: 'logx Trace 与 oops'
linkTitle: 'trace-and-oops'
description: '从 context 里补充 trace/span 字段，并配合 oops 错误使用'
weight: 4
---

## Trace context

如果你使用 OpenTelemetry，可以从 `context.Context` 里取出 trace/span ID 并打到日志字段中：

```go
package main

import (
	"context"

	"github.com/DaiYuANg/arcgo/logx"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	logger, err := logx.New(logx.WithConsole(true), logx.WithDebugLevel())
	if err != nil {
		panic(err)
	}
	defer func() { _ = logx.Close(logger) }()

	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	if err != nil {
		panic(err)
	}
	spanID, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	if err != nil {
		panic(err)
	}

	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), spanContext)

	logx.WithTraceContext(ctx, logger).Info("request accepted", "endpoint", "/api/orders")
}
```

可运行示例：

- [examples/logx/trace_context](https://github.com/DaiYuANg/arcgo/tree/main/examples/logx/trace_context)

## oops 辅助

`logx` 提供了一些与 `oops` 兼容的辅助函数：

```go
package main

import "github.com/DaiYuANg/arcgo/logx"

func main() {
	logger := logx.MustNew(logx.WithConsole(true))
	defer func() { _ = logx.Close(logger) }()

	err := logx.Oopsf("upstream %s failed", "payment")
	logx.LogOops(logger, err)
}
```
