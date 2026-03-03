---
sidebar_position: 2
---

# Usage Guide

This page explains day-to-day usage of `logx`.

## Create logger

### `New`

```go
logger, err := logx.New(
    logx.WithConsole(true),
    logx.WithLevel(logx.InfoLevel),
)
if err != nil {
    panic(err)
}
defer logger.Close()
```

### `MustNew`

```go
logger := logx.MustNew(
    logx.WithConsole(true),
    logx.WithLevel(logx.InfoLevel),
)
defer logger.Close()
```

### Presets

```go
logger := logx.MustNewDevelopment()
logger := logx.MustNewProduction()
```

## Options

```go
logx.WithConsole(true)
logx.WithFile("/var/log/app.log")
logx.WithLevel(logx.DebugLevel)
logx.WithCaller(true)
```

Rotation options:

```go
logx.WithMaxSize(100)
logx.WithMaxAge(30)
logx.WithMaxBackups(5)
logx.WithLocalTime(true)
logx.WithCompress(true)
```

## Write logs

```go
logger.Debug("Debug message")
logger.Info("Info message")
logger.Warn("Warning message")
logger.Error("Error message")
```

With fields:

```go
logger.WithField("user_id", 123).Info("User logged in")
logger.WithFields(map[string]interface{}{
    "request_id": "abc123",
    "action":     "login",
}).Info("User action")
```

With error:

```go
err := fmt.Errorf("connection failed")
logger.WithError(err).Error("Database error")
```

## Context logger

```go
requestLogger := logger.WithFields(map[string]interface{}{
    "request_id": requestID,
    "method":     r.Method,
    "path":       r.URL.Path,
})
requestLogger.Info("Request received")
```

## Global logger

```go
logger := logx.MustNew(
    logx.WithConsole(true),
    logx.WithSetGlobal(true),
)

aLog := logx.Info
_ = aLog
```

Or set manually:

```go
logger.SetGlobalLogger()
```

## Utility methods

```go
if logger.IsDebug() {
    // debug-only logic
}

level := logger.GetLevel()
levelStr := logger.GetLevelString()
_ = level
_ = levelStr
```
