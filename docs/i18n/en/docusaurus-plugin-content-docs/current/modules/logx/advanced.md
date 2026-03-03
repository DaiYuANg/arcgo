---
sidebar_position: 3
---

# Advanced Usage

This page covers advanced `logx` patterns: error tracing, slog integration, and performance tips.

## Error tracing with oops

`logx` integrates [oops](https://github.com/samber/oops) for rich error context.

```go
func doSomething() error {
    return oops.
        With("user_id", 123).
        With("operation", "database_query").
        Errorf("connection timeout")
}

if err := doSomething(); err != nil {
    logger.WithError(err).Error("Operation failed")
}
```

Error chaining:

```go
func level3() error { return oops.Errorf("level 3 error") }

func level2() error {
    return oops.With("component", "level2").Wrap(level3(), "level 2 failed")
}
```

## `slog` adapter

```go
lx := logx.MustNew(logx.WithConsole(true), logx.WithLevel(logx.DebugLevel))
defer lx.Close()

slogLogger := lx.ToSlog()
slogLogger.Info("Hello from slog", "key", "value")
```

Set as global `slog` logger:

```go
slog.SetDefault(lx.ToSlog())
```

## Output strategies

### Multi-output

```go
logger := logx.MustNew(
    logx.WithConsole(true),
    logx.WithFile("/var/log/app.log"),
)
```

### JSON-only output

```go
logger := logx.MustNew(
    logx.WithConsole(false),
    logx.WithFile("/var/log/app.log"),
)
```

## Sampling pattern

```go
type SampledLogger struct {
    logger *logx.Logger
    count  atomic.Uint64
    sample uint64
}

func (sl *SampledLogger) Info(msg string, fields ...interface{}) {
    count := sl.count.Add(1)
    if count%sl.sample == 0 {
        sl.logger.Info(msg, append(fields, "sampled_count", count)...)
    }
}
```

## Performance tips

1. Avoid expensive string building when the level is disabled.
2. Reuse derived loggers instead of recreating them repeatedly.
3. Prefer structured fields over concatenated free text.

```go
if logger.IsDebug() {
    logger.Debug("Expensive operation", "result", expensiveOperation())
}
```
