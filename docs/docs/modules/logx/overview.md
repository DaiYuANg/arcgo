---
sidebar_position: 1
---

# 概述

`logx` 是一个基于 [zerolog](https://github.com/rs/zerolog) 的高性能日志记录器，支持文件轮转和 [oops](https://github.com/samber/oops) 错误追踪集成。

## 特性

- ✅ **支持控制台和文件输出** - 可同时输出到控制台和文件
- ✅ **支持日志轮转** - 基于 [lumberjack](https://github.com/natefinch/lumberjack) 的文件轮转
- ✅ **支持错误堆栈追踪** - 集成 [oops](https://github.com/samber/oops) 错误追踪
- ✅ **支持开发/生产环境预设** - 快速配置开发或生产环境
- ✅ **简洁易用的 API** - 直观的链式调用
- ✅ **支持 slog 适配** - 可适配标准 slog 接口

## 安装

```bash
go get github.com/DaiYuANg/arcgo/logx
```

## 快速示例

### 基本用法

```go
package main

import (
    "github.com/DaiYuANg/arcgo/logx"
)

func main() {
    // 创建日志记录器
    logger := logx.MustNew(
        logx.WithConsole(true),
        logx.WithLevel(logx.InfoLevel),
    )
    defer logger.Close()

    // 记录日志
    logger.Info("Server started", "port", 8080)
    logger.Debug("Debug info", "key", "value")
    
    // 错误日志
    if err := someFunction(); err != nil {
        logger.WithError(err).Error("Operation failed")
    }
}

func someFunction() error {
    return fmt.Errorf("sample error")
}
```

### 开发环境预设

```go
package main

import (
    "github.com/DaiYuANg/arcgo/logx"
)

func main() {
    // 开发环境：console 输出 + debug 级别
    logger := logx.MustNewDevelopment()
    defer logger.Close()

    logger.Debug("Debug message")
    logger.Info("Info message")
}
```

### 生产环境预设

```go
package main

import (
    "github.com/DaiYuANg/arcgo/logx"
)

func main() {
    // 生产环境：JSON 格式 + info 级别
    logger := logx.MustNewProduction()
    defer logger.Close()

    logger.Info("Application started")
}
```

### 文件输出

```go
package main

import (
    "github.com/DaiYuANg/arcgo/logx"
)

func main() {
    logger := logx.MustNew(
        logx.WithConsole(true),
        logx.WithFile("/var/log/app.log"),
        logx.WithLevel(logx.InfoLevel),
        
        // 文件轮转配置
        logx.WithMaxSize(100),        // 每个文件最大 100MB
        logx.WithMaxAge(30),          // 保留 30 天
        logx.WithMaxBackups(5),       // 最多保留 5 个备份
        logx.WithCompress(true),      // 压缩备份
    )
    defer logger.Close()

    logger.Info("Application started")
}
```

### 添加上下文

```go
package main

import (
    "github.com/DaiYuANg/arcgo/logx"
)

func main() {
    logger := logx.MustNew(logx.WithConsole(true))
    defer logger.Close()

    // 添加单个字段
    logger.WithField("user_id", 123).Info("User logged in")

    // 添加多个字段
    logger.WithFields(map[string]interface{}{
        "user_id": 123,
        "action":  "login",
    }).Info("User action")

    // 添加错误
    err := fmt.Errorf("connection failed")
    logger.WithError(err).Error("Database error")

    // 链式调用
    logger.
        WithField("request_id", "abc123").
        WithField("user_id", 123).
        Info("Request processed")
}
```

## 日志级别

| 级别 | 说明 |
|------|------|
| `TraceLevel` | 追踪级别，最详细的日志 |
| `DebugLevel` | 调试级别，开发时使用 |
| `InfoLevel` | 信息级别，记录一般信息（默认） |
| `WarnLevel` | 警告级别，记录潜在问题 |
| `ErrorLevel` | 错误级别，记录错误 |
| `FatalLevel` | 致命级别，记录后程序退出 |
| `PanicLevel` | 恐慌级别，记录后 panic |

## 输出格式

### Console 格式（开发环境）

```
12:34:56 INF Application started
12:34:57 DBG Debug info key=value
12:34:58 ERR Operation failed error="connection refused"
```

### JSON 格式（生产环境）

```json
{"level":"info","time":"2024-01-01T12:34:56Z","message":"Application started"}
{"level":"debug","time":"2024-01-01T12:34:57Z","message":"Debug info","key":"value"}
{"level":"error","time":"2024-01-01T12:34:58Z","message":"Operation failed","error":"connection refused"}
```

## 下一步

- [使用指南](/docs/modules/logx/usage) - 学习详细用法
- [高级用法](/docs/modules/logx/advanced) - 了解错误追踪、slog 适配等
