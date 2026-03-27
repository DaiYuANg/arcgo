---
title: 'logx Trace and oops'
linkTitle: 'trace-and-oops'
description: 'Attach trace/span IDs and work with oops errors'
weight: 4
---

## Trace context

If you use OpenTelemetry, you can attach trace/span IDs from a `context.Context`:

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

Runnable example:

- [examples/logx/trace_context](https://github.com/DaiYuANg/arcgo/tree/main/examples/logx/trace_context)

## oops helpers

`logx` provides small helpers for `oops`-compatible errors:

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
