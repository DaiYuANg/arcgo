---
sidebar_position: 1
---

# Overview

`logx` is a high-performance logger built on [zerolog](https://github.com/rs/zerolog), with file rotation and [oops](https://github.com/samber/oops) integration.

## Features

- Console and file outputs
- Rotation via [lumberjack](https://github.com/natefinch/lumberjack)
- Error stack tracing via oops
- Development/production presets
- Clean API
- `slog` adapter support

## Install

```bash
go get github.com/DaiYuANg/toolkit4go/logx
```

## Quick examples

### Basic

```go
logger := logx.MustNew(
    logx.WithConsole(true),
    logx.WithLevel(logx.InfoLevel),
)
defer logger.Close()

logger.Info("Server started", "port", 8080)
logger.Debug("Debug info", "key", "value")
```

### Development preset

```go
logger := logx.MustNewDevelopment()
defer logger.Close()
```

### Production preset

```go
logger := logx.MustNewProduction()
defer logger.Close()
```

### File output + rotation

```go
logger := logx.MustNew(
    logx.WithConsole(true),
    logx.WithFile("/var/log/app.log"),
    logx.WithLevel(logx.InfoLevel),
    logx.WithMaxSize(100),
    logx.WithMaxAge(30),
    logx.WithMaxBackups(5),
    logx.WithCompress(true),
)
defer logger.Close()
```

## Levels

| Level | Description |
|------|------|
| `TraceLevel` | Most detailed |
| `DebugLevel` | Debugging |
| `InfoLevel` | General info (default) |
| `WarnLevel` | Warnings |
| `ErrorLevel` | Errors |
| `FatalLevel` | Fatal and exit |
| `PanicLevel` | Panic |

## Output formats

Console example:

```text
12:34:56 INF Application started
12:34:57 DBG Debug info key=value
12:34:58 ERR Operation failed error="connection refused"
```

JSON example:

```json
{"level":"info","time":"2024-01-01T12:34:56Z","message":"Application started"}
```

## Next

- [Usage Guide](/docs/modules/logx/usage)
- [Advanced Usage](/docs/modules/logx/advanced)
