---
sidebar_position: 2
---

# 使用指南

本文档介绍 `httpx` 的详细用法。

## 核心概念

### Endpoint（端点）

Endpoint 是 httpx 的核心抽象，代表一个或多个 HTTP 路由处理器：

```go
type UserEndpoint struct {
    httpx.BaseEndpoint
}

func (e *UserEndpoint) ListUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    e.Success(w, map[string]interface{}{
        "users": []string{"Alice", "Bob"},
    })
    return nil
}
```

### BaseEndpoint

`BaseEndpoint` 提供了常用的辅助方法：

- `Success(w, data)` - 返回成功响应
- `Error(w, code, message)` - 返回错误响应
- `NotFound(w, message)` - 返回 404
- `BadRequest(w, message)` - 返回 400
- `Param(w, r, name)` - 获取 URL 参数
- `Query(w, r, name)` - 获取查询参数
- `Header(w, r, name)` - 获取请求头

### Adapter（适配器）

适配器是 httpx 与具体 Web 框架的桥梁：

```go
// Gin 适配器
ginAdapter := gin.New()

// Fiber 适配器
fiberAdapter := fiber.New()

// Echo 适配器
echoAdapter := echo.New()

// 标准库适配器（基于 chi）
stdAdapter := std.New()
```

### Server（服务器）

Server 统一管理路由注册和服务启动：

```go
server := httpx.NewServer(
    httpx.WithAdapter(ginAdapter),
    httpx.WithBasePath("/api"),
    httpx.WithPrintRoutes(true),
)
```

## 定义 Endpoint

### 基本 Endpoint

```go
package main

import (
    "context"
    "net/http"
    "github.com/DaiYuANg/arcgo/httpx"
)

type UserEndpoint struct {
    httpx.BaseEndpoint
}

// GET /users
func (e *UserEndpoint) ListUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    e.Success(w, map[string]interface{}{
        "users": []string{"Alice", "Bob", "Charlie"},
    })
    return nil
}

// GET /users/:id
func (e *UserEndpoint) GetUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    id := e.Param(w, r, "id")
    e.Success(w, map[string]interface{}{
        "id":   id,
        "name": "User " + id,
    })
    return nil
}

// POST /users
func (e *UserEndpoint) CreateUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    var data map[string]interface{}
    if err := e.ParseJSON(r, &data); err != nil {
        return e.BadRequest(w, "Invalid JSON")
    }
    
    e.Success(w, map[string]interface{}{
        "id":   1,
        "name": data["name"],
    })
    return nil
}
```

### 带依赖注入的 Endpoint

```go
type UserEndpoint struct {
    httpx.BaseEndpoint
    userService *UserService
    logger      *logx.Logger
}

func NewUserEndpoint(userService *UserService, logger *logx.Logger) *UserEndpoint {
    return &UserEndpoint{
        userService: userService,
        logger:      logger,
    }
}

func (e *UserEndpoint) ListUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    e.logger.Info("Listing users")
    users, err := e.userService.List(ctx)
    if err != nil {
        return e.Error(w, 500, "Failed to list users")
    }
    e.Success(w, users)
    return nil
}
```

### 使用 Huma OpenAPI

```go
type UserEndpoint struct {
    httpx.BaseEndpoint
}

// huma:ListUsers
// @Summary List users
// @Description Get a list of users
// @Tags users
// @Success 200 {object} []User
func (e *UserEndpoint) ListUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    // ...
}
```

## 创建服务器

### 基本用法

```go
server := httpx.NewServer(
    httpx.WithAdapter(ginAdapter),
)
```

### 配置选项

```go
server := httpx.NewServer(
    httpx.WithAdapter(ginAdapter),      // 适配器（必需）
    httpx.WithBasePath("/api"),         // 基础路径
    httpx.WithPrintRoutes(true),        // 打印路由信息
    httpx.WithLogger(logger),           // 自定义日志
)
```

### 注册 Endpoint

```go
// 注册单个 Endpoint
_ = server.Register(&UserEndpoint{})

// 注册多个 Endpoint
_ = server.Register(&UserEndpoint{}, &ProductEndpoint{}, &OrderEndpoint{})

// 链式注册
server.
    Register(&UserEndpoint{}).
    Register(&ProductEndpoint{}).
    Register(&OrderEndpoint{})
```

## 路由规则

### HTTP 方法映射

httpx 根据方法名自动映射 HTTP 方法：

| 方法名前缀 | HTTP 方法 |
|-----------|----------|
| `Get` | GET |
| `Post` | POST |
| `Put` | PUT |
| `Delete` | DELETE |
| `Patch` | PATCH |
| `Head` | HEAD |
| `Options` | OPTIONS |

### 路由路径

路由路径由 Endpoint 类型名和方法名生成：

```go
type UserEndpoint struct{}

// GET /users
func (e *UserEndpoint) ListUsers() {}

// GET /users/:id
func (e *UserEndpoint) GetUser() {}

// POST /users
func (e *UserEndpoint) CreateUser() {}

// PUT /users/:id
func (e *UserEndpoint) UpdateUser() {}

// DELETE /users/:id
func (e *UserEndpoint) DeleteUser() {}
```

### 自定义路由

```go
type UserEndpoint struct {
    httpx.BaseEndpoint
}

func (e *UserEndpoint) Routes() []httpx.Route {
    return []httpx.Route{
        {Method: "GET", Path: "/users", Handler: e.ListUsers},
        {Method: "GET", Path: "/users/:id", Handler: e.GetUser},
        {Method: "POST", Path: "/users", Handler: e.CreateUser},
    }
}
```

## 请求处理

### 获取参数

```go
// URL 参数：/users/:id
func (e *UserEndpoint) GetUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    id := e.Param(w, r, "id")
    // ...
}

// 查询参数：/users?page=1&size=10
func (e *UserEndpoint) ListUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    page := e.QueryInt(w, r, "page", 1)
    size := e.QueryInt(w, r, "size", 10)
    // ...
}

// 请求头
func (e *UserEndpoint) GetInfo(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    auth := e.Header(w, r, "Authorization")
    // ...
}
```

### 解析请求体

```go
// JSON
func (e *UserEndpoint) CreateUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    var data CreateUserInput
    if err := e.ParseJSON(r, &data); err != nil {
        return e.BadRequest(w, "Invalid JSON")
    }
    // ...
}

// Form
func (e *UserEndpoint) UpdateProfile(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    name := r.FormValue("name")
    email := r.FormValue("email")
    // ...
}
```

### 响应处理

```go
// 成功响应
e.Success(w, data)
e.SuccessWithCode(w, 201, data)

// 错误响应
e.Error(w, 500, "Internal error")
e.BadRequest(w, "Invalid input")
e.NotFound(w, "User not found")
e.Unauthorized(w, "Unauthorized")
e.Forbidden(w, "Forbidden")
```

## 错误日志

`httpx` 在以下内部错误场景会自动打印 `slog` 错误日志：

- `BaseEndpoint.JSON/Success/Error` 在 JSON 序列化失败或响应写回失败时
- `huma.Service.RegisterHandler` 在 OpenAPI JSON 或 Docs HTML 写回失败时

默认使用 `slog.Default()`。你可以通过 `slog.SetDefault(...)` 统一接入自己的日志系统：

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))
slog.SetDefault(logger)
```

如果你使用 `huma.Service`，也可以单独注入 logger：

```go
svc := huma.NewService(api, "My API", "1.0.0", "API docs").
    WithLogger(logger)
```

## 完整示例

```go
package main

import (
    "context"
    "net/http"
    "github.com/DaiYuANg/arcgo/httpx"
    "github.com/DaiYuANg/arcgo/httpx/adapter/gin"
    "github.com/DaiYuANg/arcgo/logx"
)

type UserService struct {
    logger *logx.Logger
}

func (s *UserService) List(ctx context.Context) ([]map[string]interface{}, error) {
    return []map[string]interface{}{
        {"id": 1, "name": "Alice"},
        {"id": 2, "name": "Bob"},
    }, nil
}

type UserEndpoint struct {
    httpx.BaseEndpoint
    userService *UserService
    logger      *logx.Logger
}

func NewUserEndpoint(userService *UserService, logger *logx.Logger) *UserEndpoint {
    return &UserEndpoint{
        userService: userService,
        logger:      logger,
    }
}

// GET /api/users
func (e *UserEndpoint) ListUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    e.logger.Info("Listing users")
    users, err := e.userService.List(ctx)
    if err != nil {
        return e.Error(w, 500, "Failed to list users")
    }
    e.Success(w, users)
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

// POST /api/users
func (e *UserEndpoint) CreateUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    var data map[string]interface{}
    if err := e.ParseJSON(r, &data); err != nil {
        return e.BadRequest(w, "Invalid JSON")
    }
    e.SuccessWithCode(w, 201, data)
    return nil
}

func main() {
    // 创建日志
    logger := logx.MustNew(logx.WithConsole(true))
    defer logger.Close()

    // 创建服务
    userService := &UserService{logger: logger}

    // 创建适配器
    ginAdapter := gin.New()
    ginAdapter.Engine().Use(gin.Logger(), gin.Recovery())

    // 启用 OpenAPI
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

    // 注册 Endpoint
    _ = server.Register(NewUserEndpoint(userService, logger))

    // 启动服务
    server.ListenAndServe(":8080")
}
```
