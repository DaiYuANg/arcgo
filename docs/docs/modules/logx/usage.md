---
sidebar_position: 2
---

# 使用指南

本文档介绍 `logx` 的详细用法。

## 创建日志记录器

### 使用 New 函数

```go
package main

import (
    "github.com/DaiYuANg/arcgo/logx"
)

func main() {
    logger, err := logx.New(
        logx.WithConsole(true),
        logx.WithLevel(logx.InfoLevel),
    )
    if err != nil {
        panic(err)
    }
    defer logger.Close()

    logger.Info("Application started")
}
```

### 使用 MustNew 函数

```go
// 创建失败会 panic
logger := logx.MustNew(
    logx.WithConsole(true),
    logx.WithLevel(logx.InfoLevel),
)
defer logger.Close()
```

### 使用预设

```go
// 开发环境预设
logger := logx.MustNewDevelopment()
defer logger.Close()

// 生产环境预设
logger := logx.MustNewProduction()
defer logger.Close()
```

## 配置选项

### WithConsole

启用控制台输出：

```go
logx.WithConsole(true)
```

### WithFile

启用文件输出：

```go
logx.WithFile("/var/log/app.log")
```

### WithLevel

设置日志级别：

```go
logx.WithLevel(logx.DebugLevel)
logx.WithLevel(logx.InfoLevel)
logx.WithLevel(logx.WarnLevel)
logx.WithLevel(logx.ErrorLevel)
```

### WithCaller

启用调用者信息：

```go
logx.WithCaller(true)
```

### 文件轮转配置

```go
logx.WithFile("/var/log/app.log")
logx.WithMaxSize(100)       // 每个文件最大 100MB
logx.WithMaxAge(30)         // 保留 30 天
logx.WithMaxBackups(5)      // 最多保留 5 个备份
logx.WithLocalTime(true)    // 使用本地时间
logx.WithCompress(true)     // 压缩备份
```

## 记录日志

### 基本方法

```go
logger.Debug("Debug message")
logger.Info("Info message")
logger.Warn("Warning message")
logger.Error("Error message")
logger.Fatal("Fatal message")
logger.Panic("Panic message")
```

### 添加字段

```go
// 添加单个字段
logger.Debug("Processing request", "request_id", "abc123")

// 添加多个字段（使用 map）
logger.Info("User action", map[string]interface{}{
    "user_id": 123,
    "action":  "login",
    "ip":      "192.168.1.1",
})
```

### 使用 WithField

```go
// 添加单个字段
logger.WithField("user_id", 123).Info("User logged in")

// 链式调用
logger.
    WithField("request_id", "abc123").
    WithField("user_id", 123).
    Info("Request processed")
```

### 使用 WithFields

```go
logger.WithFields(map[string]interface{}{
    "user_id":   123,
    "request_id": "abc123",
    "action":    "login",
}).Info("User action")
```

### 记录错误

```go
err := fmt.Errorf("connection failed")

// 添加错误
logger.WithError(err).Error("Database error")

// 添加错误和额外字段
logger.WithError(err).
    WithField("operation", "query").
    Error("Database operation failed")
```

## 日志上下文

### 创建子日志记录器

```go
// 基于现有 logger 创建带上下文的子 logger
requestLogger := logger.WithField("request_id", "abc123")

// 后续使用
requestLogger.Info("Request started")
requestLogger.Info("Request completed")
```

### 在 HTTP 请求中使用

```go
type Handler struct {
    logger *logx.Logger
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 为每个请求创建带上下文的 logger
    requestID := generateRequestID()
    requestLogger := h.logger.WithFields(map[string]interface{}{
        "request_id": requestID,
        "method":     r.Method,
        "path":       r.URL.Path,
        "remote_addr": r.RemoteAddr,
    })

    requestLogger.Info("Request received")
    
    // 处理请求
    // ...
    
    requestLogger.Info("Request completed")
}
```

## 全局日志记录器

### 设置为全局

```go
logger := logx.MustNew(
    logx.WithConsole(true),
    logx.WithSetGlobal(true), // 设置为全局
)
defer logger.Close()

// 使用全局 logger
logx.Debug("Debug message")
logx.Info("Info message")
```

### 手动设置全局

```go
logger := logx.MustNew(logx.WithConsole(true))
defer logger.Close()

// 设置为全局
logger.SetGlobalLogger()

// 使用全局 logger
logx.Info("Application started")
```

## 便捷方法

### 检查日志级别

```go
if logger.IsDebug() {
    // 执行调试逻辑
}

if logger.IsTrace() {
    // 执行追踪逻辑
}

if logger.IsInfo() {
    // 执行信息逻辑
}
```

### 获取日志级别

```go
level := logger.GetLevel()
levelStr := logger.GetLevelString()
```

## 完整示例

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "time"
    "github.com/DaiYuANg/arcgo/logx"
)

type Server struct {
    logger *logx.Logger
}

func NewServer(logger *logx.Logger) *Server {
    return &Server{logger: logger}
}

func (s *Server) Start(addr string) error {
    s.logger.Info("Starting server", "address", addr)
    
    mux := http.NewServeMux()
    mux.HandleFunc("/health", s.healthHandler)
    mux.HandleFunc("/api/users", s.usersHandler)
    
    server := &http.Server{
        Addr:         addr,
        Handler:      mux,
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 10 * time.Second,
    }
    
    s.logger.Info("Server started", "address", addr)
    return server.ListenAndServe()
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
    requestLogger := s.logger.WithFields(map[string]interface{}{
        "method": r.Method,
        "path":   r.URL.Path,
    })
    
    requestLogger.Debug("Health check requested")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
}

func (s *Server) usersHandler(w http.ResponseWriter, r *http.Request) {
    requestLogger := s.logger.WithFields(map[string]interface{}{
        "method": r.Method,
        "path":   r.URL.Path,
    })
    
    requestLogger.Info("Users request received")
    
    // 模拟处理
    users := []map[string]interface{}{
        {"id": 1, "name": "Alice"},
        {"id": 2, "name": "Bob"},
    }
    
    requestLogger.Info("Users returned", "count", len(users))
    
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprintf(w, `{"users":%v}`, users)
}

func main() {
    // 创建日志记录器
    logger := logx.MustNew(
        logx.WithConsole(true),
        logx.WithFile("/var/log/app.log"),
        logx.WithLevel(logx.InfoLevel),
        logx.WithCaller(true),
        logx.WithMaxSize(100),
        logx.WithMaxAge(30),
        logx.WithMaxBackups(5),
    )
    defer logger.Close()

    logger.Info("Application starting")

    // 启动服务
    server := NewServer(logger)
    if err := server.Start(":8080"); err != nil {
        logger.WithError(err).Fatal("Server failed")
    }
}
```
