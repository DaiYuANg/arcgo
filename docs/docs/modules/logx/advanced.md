---
sidebar_position: 3
---

# 高级用法

本文档介绍 `logx` 的高级用法，包括错误追踪、slog 适配等。

## 错误追踪（oops 集成）

`logx` 集成了 [oops](https://github.com/samber/oops)，提供强大的错误追踪功能。

### 基本使用

```go
package main

import (
    "github.com/DaiYuANg/arcgo/logx"
    "github.com/samber/oops"
)

func main() {
    logger := logx.MustNew(logx.WithConsole(true))
    defer logger.Close()

    if err := doSomething(); err != nil {
        logger.WithError(err).Error("Operation failed")
    }
}

func doSomething() error {
    // 使用 oops 包装错误
    return oops.
        With("user_id", 123).
        With("operation", "database_query").
        Errorf("connection timeout")
}
```

### 错误堆栈追踪

```go
func doSomething() error {
    return oops.
        With("user_id", 123).
        With("operation", "database_query").
        With("retry_count", 3).
        Errorf("connection timeout")
}

func handleError(err error, logger *logx.Logger) {
    // oops 错误会自动包含堆栈信息
    logger.WithError(err).Error("Operation failed")
}
```

### 错误上下文

```go
func processUser(userID int) error {
    return oops.
        With("user_id", userID).
        With("operation", "process_user").
        With("timestamp", time.Now()).
        Errorf("failed to process user")
}

func getUserProfile(userID string) (map[string]interface{}, error) {
    profile, err := fetchProfile(userID)
    if err != nil {
        return nil, oops.
            With("user_id", userID).
            With("operation", "get_profile").
            With("source", "database").
            Wrap(err, "failed to fetch profile")
    }
    return profile, nil
}
```

### 错误链

```go
func level3() error {
    return oops.Errorf("level 3 error")
}

func level2() error {
    err := level3()
    return oops.
        With("component", "level2").
        Wrap(err, "level 2 failed")
}

func level1() error {
    err := level2()
    return oops.
        With("component", "level1").
        Wrap(err, "level 1 failed")
}

func main() {
    logger := logx.MustNew(logx.WithConsole(true))
    defer logger.Close()

    if err := level1(); err != nil {
        logger.WithError(err).Error("Application error")
    }
}
```

## slog 适配

`logx` 可以适配标准 `log/slog` 接口。

### 创建 slog 日志记录器

```go
package main

import (
    "log/slog"
    "github.com/DaiYuANg/arcgo/logx"
)

func main() {
    // 创建 logx logger
    lx := logx.MustNew(
        logx.WithConsole(true),
        logx.WithLevel(logx.DebugLevel),
    )
    defer lx.Close()

    // 转换为 slog 接口
    slogLogger := lx.ToSlog()

    // 使用 slog API
    slogLogger.Info("Hello from slog", "key", "value")
    slogLogger.Error("Error from slog", "error", err)
}
```

### 与标准库集成

```go
package main

import (
    "log/slog"
    "net/http"
    "github.com/DaiYuANg/arcgo/logx"
)

func main() {
    lx := logx.MustNew(logx.WithConsole(true))
    defer lx.Close()

    // 设置全局 slog
    slog.SetDefault(lx.ToSlog())

    // 现在标准库的 slog 调用会使用 logx
    slog.Info("Application started")
    
    // HTTP 服务器会使用 slog
    http.ListenAndServe(":8080", nil)
}
```

## 自定义输出

### 多输出

```go
logger := logx.MustNew(
    logx.WithConsole(true),
    logx.WithFile("/var/log/app.log"),
)
defer logger.Close()

// 日志会同时输出到控制台和文件
logger.Info("Application started")
```

### 仅文件输出

```go
logger := logx.MustNew(
    logx.WithConsole(false),
    logx.WithFile("/var/log/app.log"),
)
defer logger.Close()
```

### 仅控制台输出

```go
logger := logx.MustNew(
    logx.WithConsole(true),
    logx.WithFile(""), // 不设置文件
)
defer logger.Close()
```

## 自定义格式化

### 控制台格式化

```go
logger := logx.MustNew(
    logx.WithConsole(true),
    logx.WithTimeFormat("2006-01-02 15:04:05"),
    logx.WithNoColor(false), // 启用颜色
)
defer logger.Close()
```

### JSON 格式化

```go
logger := logx.MustNew(
    logx.WithConsole(false), // 禁用控制台（JSON 格式）
    logx.WithFile("/var/log/app.log"),
)
defer logger.Close()
```

## 采样日志

对于高频日志，可以使用采样避免日志爆炸：

```go
package main

import (
    "sync/atomic"
    "github.com/DaiYuANg/arcgo/logx"
)

type SampledLogger struct {
    logger *logx.Logger
    count  atomic.Uint64
    sample uint64
}

func NewSampledLogger(logger *logx.Logger, sample uint64) *SampledLogger {
    return &SampledLogger{
        logger: logger,
        sample: sample,
    }
}

func (sl *SampledLogger) Info(msg string, fields ...interface{}) {
    count := sl.count.Add(1)
    if count%sl.sample == 0 {
        sl.logger.Info(msg, append(fields, "sampled_count", count)...)
    }
}

func main() {
    logger := logx.MustNew(logx.WithConsole(true))
    defer logger.Close()

    sampled := NewSampledLogger(logger, 1000) // 每 1000 条记录一次

    for i := 0; i < 10000; i++ {
        sampled.Info("Request processed", "request_id", i)
    }
}
```

## 性能优化

### 避免不必要的日志

```go
// ❌ 不推荐：即使不输出也会构造日志
logger.Debug("Expensive operation: " + expensiveOperation())

// ✅ 推荐：先检查级别
if logger.IsDebug() {
    logger.Debug("Expensive operation: " + expensiveOperation())
}
```

### 复用日志记录器

```go
// ❌ 不推荐：每次都创建新的 logger
func handleRequest(logger *logx.Logger, requestID string) {
    requestLogger := logger.WithField("request_id", requestID)
    // ...
}

// ✅ 推荐：复用 logger
type Handler struct {
    baseLogger *logx.Logger
}

func (h *Handler) handleRequest(requestID string) {
    requestLogger := h.baseLogger.WithField("request_id", requestID)
    // ...
}
```

## 测试

### 测试日志输出

```go
package main

import (
    "bytes"
    "testing"
    "github.com/rs/zerolog"
    "github.com/DaiYuANg/arcgo/logx"
)

func TestLogger(t *testing.T) {
    // 创建测试 logger
    var buf bytes.Buffer
    logger := logx.MustNew(
        logx.WithConsole(false),
        logx.WithLevel(logx.DebugLevel),
    )
    
    // 重定向输出到 buffer
    logger.Logger().Output(&buf)
    
    // 测试
    logger.Info("Test message")
    
    // 验证输出
    if !bytes.Contains(buf.Bytes(), []byte("Test message")) {
        t.Error("Expected log message")
    }
}
```

## 最佳实践

### 1. 使用结构化日志

```go
// ✅ 推荐
logger.Info("User logged in", "user_id", 123, "ip", "192.168.1.1")

// ❌ 不推荐
logger.Info("User 123 logged in from 192.168.1.1")
```

### 2. 使用合适的日志级别

```go
// Debug: 详细的调试信息
logger.Debug("SQL query", "query", sql, "params", params)

// Info: 一般信息
logger.Info("Server started", "address", addr)

// Warn: 警告
logger.Warn("High memory usage", "percent", 85)

// Error: 错误
logger.WithError(err).Error("Database connection failed")
```

### 3. 为日志添加上下文

```go
// 在请求开始时创建带上下文的 logger
requestLogger := logger.WithFields(map[string]interface{}{
    "request_id": requestID,
    "method":     r.Method,
    "path":       r.URL.Path,
})

// 在整个请求处理过程中使用
requestLogger.Info("Request received")
// ...
requestLogger.Info("Request completed")
```

### 4. 记录错误时包含堆栈

```go
// ✅ 推荐：使用 oops
import "github.com/samber/oops"

func doSomething() error {
    return oops.
        With("user_id", 123).
        With("operation", "database").
        Errorf("connection failed")
}

// 记录时包含完整堆栈
logger.WithError(err).Error("Operation failed")
```

### 5. 生产环境使用 JSON 格式

```go
// 生产环境
logger := logx.MustNew(
    logx.WithConsole(false), // JSON 格式
    logx.WithFile("/var/log/app.log"),
    logx.WithLevel(logx.InfoLevel),
)
```
