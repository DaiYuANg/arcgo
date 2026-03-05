---
sidebar_position: 2
---

# 快速开始

本指南将帮助你快速开始使用 arcgo。

## 环境要求

- Go 1.25.0 或更高版本
- Node.js 20.0+（仅用于运行文档站点）

## 安装

### 安装 Go 模块

根据你的需求选择安装：

```bash
# 配置加载模块
go get github.com/DaiYuANg/arcgo/configx

# 日志记录器
go get github.com/DaiYuANg/arcgo/logx

# HTTP 框架适配器（选择你需要的框架）
go get github.com/DaiYuANg/arcgo/httpx/adapter/gin
go get github.com/DaiYuANg/arcgo/httpx/adapter/fiber
go get github.com/DaiYuANg/arcgo/httpx/adapter/echo
go get github.com/DaiYuANg/arcgo/httpx/adapter/std
```

### 完整安装

安装所有模块：

```bash
go get github.com/DaiYuANg/arcgo/...
```

## 快速示例

### 1. 使用 configx 加载配置

创建 `config.yaml`：

```yaml
app:
  name: my-application
  port: 8080
  debug: true

database:
  host: localhost
  port: 5432
  user: admin
  password: secret
```

创建 `.env`：

```env
APP_NAME=my-app
APP_PORT=3000
DATABASE_HOST=db.example.com
```

编写 Go 代码：

```go
package main

import (
    "fmt"
    "github.com/DaiYuANg/arcgo/configx"
)

type Config struct {
    App struct {
        Name  string `mapstructure:"name" validate:"required"`
        Port  int    `mapstructure:"port" validate:"required"`
        Debug bool   `mapstructure:"debug"`
    } `mapstructure:"app"`
    Database struct {
        Host     string `mapstructure:"host" validate:"required,hostname"`
        Port     int    `mapstructure:"port" validate:"required"`
        User     string `mapstructure:"user"`
        Password string `mapstructure:"password"`
    } `mapstructure:"database"`
}

func main() {
    var cfg Config

    err := configx.Load(&cfg,
        configx.WithDotenv(),
        configx.WithFiles("config.yaml"),
        configx.WithEnvPrefix("APP"),
        configx.WithValidateLevel(configx.ValidateLevelRequired),
    )
    if err != nil {
        panic(err)
    }

    fmt.Printf("App: %s:%d (debug=%v)\n", cfg.App.Name, cfg.App.Port, cfg.App.Debug)
    fmt.Printf("Database: %s:%d\n", cfg.Database.Host, cfg.Database.Port)
}
```

### 2. 使用 logx 记录日志

```go
package main

import (
    "github.com/DaiYuANg/arcgo/logx"
)

func main() {
    // 创建日志记录器
    logger := logx.MustNew(
        logx.WithConsole(true),
        logx.WithFile("/var/log/app.log"),
        logx.WithLevel(logx.InfoLevel),
        logx.WithCaller(true),
    )
    defer logger.Close()

    // 记录日志
    logger.Info("Server started", "port", 8080)
    logger.Debug("Debug info", "key", "value")
    
    // 使用 WithField 添加上下文
    requestLogger := logger.WithField("request_id", "12345")
    requestLogger.Info("Processing request")
    
    // 错误日志
    if err := someFunction(); err != nil {
        logger.WithError(err).Error("Operation failed")
    }
}

func someFunction() error {
    return fmt.Errorf("sample error")
}
```

### 3. 使用 httpx 创建 HTTP 服务

```go
package main

import (
    "context"
    "net/http"
    "github.com/DaiYuANg/arcgo/httpx"
    "github.com/DaiYuANg/arcgo/httpx/adapter/gin"
)

// 定义端点
type UserEndpoint struct {
    httpx.BaseEndpoint
}

// GET /api/users
func (e *UserEndpoint) ListUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    e.Success(w, map[string]interface{}{
        "users": []string{"Alice", "Bob", "Charlie"},
    })
    return nil
}

// GET /api/users/:id
func (e *UserEndpoint) GetUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    id := e.Param(w, r, "id")
    e.Success(w, map[string]interface{}{
        "id":   id,
        "name": "User " + id,
    })
    return nil
}

func main() {
    // 创建 Gin 适配器
    ginAdapter := gin.New()
    
    // 使用 Gin 原生中间件
    ginAdapter.Engine().Use(
        gin.Logger(),
        gin.Recovery(),
    )
    
    // 启用 OpenAPI 文档
    ginAdapter.WithHuma(httpx.ToAdapterHumaOptions(httpx.HumaOptions{
        Enabled: true,
        Title:   "My API",
        Version: "1.0.0",
    }))
    
    // 创建服务器
    server := httpx.NewServer(
        httpx.WithAdapter(ginAdapter),
        httpx.WithBasePath("/api"),
        httpx.WithPrintRoutes(true),
    )
    
    // 注册端点
    _ = server.Register(&UserEndpoint{})
    
    // 启动服务
    server.ListenAndServe(":8080")
}
```

访问 `http://localhost:8080/docs` 查看 OpenAPI 文档。

## 完整示例

下面是一个结合所有模块的完整示例：

```go
package main

import (
    "context"
    "net/http"
    "github.com/DaiYuANg/arcgo/configx"
    "github.com/DaiYuANg/arcgo/logx"
    "github.com/DaiYuANg/arcgo/httpx"
    "github.com/DaiYuANg/arcgo/httpx/adapter/gin"
)

type Config struct {
    App struct {
        Name  string `mapstructure:"name"`
        Port  int    `mapstructure:"port"`
    } `mapstructure:"app"`
}

type HealthEndpoint struct {
    httpx.BaseEndpoint
    logger *logx.Logger
}

func (e *HealthEndpoint) Check(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    e.logger.Info("Health check requested")
    e.Success(w, map[string]interface{}{
        "status": "healthy",
    })
    return nil
}

func main() {
    // 1. 加载配置
    var cfg Config
    err := configx.Load(&cfg,
        configx.WithDotenv(),
        configx.WithFiles("config.yaml"),
    )
    if err != nil {
        panic(err)
    }

    // 2. 创建日志记录器
    logger := logx.MustNew(
        logx.WithConsole(true),
        logx.WithLevel(logx.InfoLevel),
    )
    defer logger.Close()

    logger.Info("Starting application", "name", cfg.App.Name)

    // 3. 创建 HTTP 服务
    ginAdapter := gin.New()
    ginAdapter.Engine().Use(gin.Logger(), gin.Recovery())
    
    server := httpx.NewServer(
        httpx.WithAdapter(ginAdapter),
        httpx.WithBasePath("/api"),
    )
    
    // 4. 注册端点
    healthEndpoint := &HealthEndpoint{logger: logger}
    _ = server.Register(healthEndpoint)
    
    // 5. 启动服务
    addr := fmt.Sprintf(":%d", cfg.App.Port)
    logger.Info("Server listening", "address", addr)
    server.ListenAndServe(addr)
}
```

## 下一步

- 深入学习 [configx](/docs/modules/configx/overview)
- 探索 [httpx](/docs/modules/httpx/overview)
- 了解 [logx](/docs/modules/logx/overview)
