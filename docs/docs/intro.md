---
sidebar_position: 1
---

# 欢迎使用 arcgo

**arcgo** 是一套简洁高效的 Go 工具库，旨在提供常用的基础设施组件，减少样板代码，让你专注于业务逻辑。

## 🎯 设计目标

- **简洁易用** - 提供直观的 API，减少学习成本
- **灵活可扩展** - 模块化设计，按需引入
- **生产就绪** - 经过实践验证，稳定可靠
- **生态友好** - 与主流 Go 库无缝集成

## 📦 核心模块

### 1. configx - 配置加载

基于 [koanf](https://github.com/knadh/koanf) 的配置加载库，支持多种配置源和结构体验证。

**主要特性：**
- 支持 `.env` 文件、配置文件（YAML/JSON/TOML）、环境变量
- 可配置加载优先级
- 支持默认值设置
- 基于 validator 的结构体验证

**快速示例：**

```go
package main

import (
    "fmt"
    "github.com/DaiYuANg/arcgo/configx"
)

type Config struct {
    Name  string `mapstructure:"name" validate:"required"`
    Port  int    `mapstructure:"port" validate:"required,min=1024,max=65535"`
    Debug bool   `mapstructure:"debug"`
}

func main() {
    var cfg Config
    err := configx.Load(&cfg,
        configx.WithDotenv(),
        configx.WithFiles("config.yaml"),
        configx.WithEnvPrefix("APP"),
    )
    if err != nil {
        panic(err)
    }
    fmt.Printf("Config: %+v\n", cfg)
}
```

[查看详细文档 →](/docs/modules/configx/overview)

---

### 2. httpx - HTTP 框架适配器

统一的 HTTP 框架适配器层，支持 Gin、Fiber、Echo、Chi 等主流框架。

**主要特性：**
- 按需引入 - 每个适配器独立子包
- 原生中间件支持 - 直接使用框架生态
- 统一接口 - 无缝切换不同框架
- Huma OpenAPI 集成 - 自动生成 API 文档

**快速示例：**

```go
package main

import (
    "context"
    "net/http"
    "github.com/DaiYuANg/arcgo/httpx"
    "github.com/DaiYuANg/arcgo/httpx/adapter/gin"
)

type UserEndpoint struct {
    httpx.BaseEndpoint
}

func (e *UserEndpoint) ListUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    e.Success(w, map[string]interface{}{
        "users": []string{"Alice", "Bob", "Charlie"},
    })
    return nil
}

func main() {
    ginAdapter := gin.New()
    ginAdapter.Engine().Use(gin.Logger(), gin.Recovery())
    
    server := httpx.NewServer(httpx.WithAdapter(ginAdapter))
    _ = server.Register(&UserEndpoint{})
    server.ListenAndServe(":8080")
}
```

[查看详细文档 →](/docs/modules/httpx/overview)

---

### 3. logx - 日志记录器

基于 zerolog 的高性能日志记录器，支持文件轮转和错误追踪。

**主要特性：**
- 支持控制台和文件输出
- 支持日志轮转（基于 lumberjack）
- 支持错误堆栈追踪（oops 集成）
- 开发/生产环境预设

**快速示例：**

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
    )
    defer logger.Close()

    logger.Info("Server started", "port", 8080)
    logger.Error("Something failed", "error", err)
}
```

[查看详细文档 →](/docs/modules/logx/overview)

---

## 🚀 开始使用

访问 [快速开始](/docs/quick-start) 了解更多。
