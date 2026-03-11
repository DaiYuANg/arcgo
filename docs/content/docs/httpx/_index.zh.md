---
title: 'httpx'
linkTitle: 'httpx'
description: '多框架统一强类型 HTTP 路由'
weight: 5
---

## httpx

`httpx` 是构建在 Huma 之上的轻量级 HTTP 服务组织层。

## Roadmap

- 模块路线图见：[httpx roadmap](./roadmap)
- 全局路线图见：[ArcGo roadmap](../roadmap)

## 你得到什么

- 跨适配器的统一类型化路由注册（`Get`、`Post`、`Put`、`Patch`、`Delete`...）
- 基于适配器的运行时集成（`std`、`gin`、`echo`、`fiber`）
- 一流的 OpenAPI 和文档控制
- 类型化 Server-Sent Events（SSE）路由注册（`GetSSE`、`GroupGetSSE`）
- 基于策略的路由能力（`RouteWithPolicies`、`GroupRouteWithPolicies`）
- 条件请求处理（`If-Match`、`If-None-Match`、`If-Modified-Since`、`If-Unmodified-Since`）
- 直接 Huma 逃生舱（`HumaAPI`、`OpenAPI`、`ConfigureOpenAPI`）
- 组级 Huma 中间件和操作自定义
- 通过 `go-playground/validator` 进行可选请求验证
- 用于测试和诊断的路由 introspection API

## 定位

`httpx` 不是重型 web 框架，也不打算替换 Huma。
它提供稳定的 server/group/endpoint API 表面，同时保留对 Huma 高级功能的直接访问。

职责划分如下：

- `Huma`: 类型化操作、schema、OpenAPI、文档、中间件模型
- `adapter/*`: 运行时、路由器集成、原生中间件生态系统
- `httpx`: 统一服务组织 API 和 Huma 能力暴露

## 最小设置

```go
package main

import (
    "context"

    "github.com/DaiYuANg/arcgo/httpx"
    "github.com/DaiYuANg/arcgo/httpx/adapter/std"
    "github.com/go-chi/chi/v5/middleware"
)

type HealthOutput struct {
    Body struct {
        Status string `json:"status"`
    }
}

func main() {
    a := std.New()
    a.Router().Use(middleware.Logger, middleware.Recoverer)

    s := httpx.NewServer(
        httpx.WithAdapter(a),
        httpx.WithBasePath("/api"),
        httpx.WithOpenAPIInfo("My API", "1.0.0", "Service API"),
    )

    _ = httpx.Get(s, "/health", func(ctx context.Context, input *struct{}) (*HealthOutput, error) {
        out := &HealthOutput{}
        out.Body.Status = "ok"
        return out, nil
    })

    _ = s.ListenAndServe(":8080")
}
```

## 核心 API

### Server

- `NewServer(...)`
- `WithAdapter(...)`
- `WithBasePath(...)`
- `WithValidation()` / `WithValidator(...)`
- `WithPanicRecover(...)`
- `WithAccessLog(...)`
- `HumaAPI()`
- `OpenAPI()`
- `ConfigureOpenAPI(...)`
- `PatchOpenAPI(...)`
- `UseHumaMiddleware(...)`

### 文档 / OpenAPI

构建时文档配置：

```go
s := httpx.NewServer(
    httpx.WithDocs(httpx.DocsOptions{
        Enabled:     true,
        DocsPath:    "/reference",
        OpenAPIPath: "/spec",
        SchemasPath: "/schemas",
        Renderer:    httpx.DocsRendererScalar,
    }),
)
```

运行时文档配置：

```go
s.ConfigureDocs(func(d *httpx.DocsOptions) {
    d.DocsPath = "/docs/internal"
    d.OpenAPIPath = "/openapi/internal"
})
```

OpenAPI 打补丁：

```go
s.ConfigureOpenAPI(func(doc *huma.OpenAPI) {
    doc.Tags = append(doc.Tags, &huma.Tag{Name: "internal"})
})
```

说明：

- `WithOpenAPIInfo(...)` 和 `WithOpenAPIDocs(...)` 仍然有效。
- `ConfigureDocs(...)` 现在也更新适配器管理的文档路由。
- 支持的内置渲染器：
  - `httpx.DocsRendererStoplightElements`
  - `httpx.DocsRendererScalar`
  - `httpx.DocsRendererSwaggerUI`

### Security / Components / 全局参数

```go
s := httpx.NewServer(
    httpx.WithSecurity(httpx.SecurityOptions{
        Schemes: map[string]*huma.SecurityScheme{
            "bearerAuth": {
                Type:   "http",
                Scheme: "bearer",
            },
        },
        Requirements: []map[string][]string{
            {"bearerAuth": {}},
        },
    }),
)

s.RegisterComponentParameter("Locale", &huma.Param{
    Name: "locale",
    In:   "query",
    Schema: &huma.Schema{Type: "string"},
})

s.RegisterGlobalHeader(&huma.Param{
    Name:   "X-Request-Id",
    In:     "header",
    Schema: &huma.Schema{Type: "string"},
})
```

可用 API：

- `RegisterSecurityScheme(...)`
- `SetDefaultSecurity(...)`
- `RegisterComponentParameter(...)`
- `RegisterComponentHeader(...)`
- `RegisterGlobalParameter(...)`
- `RegisterGlobalHeader(...)`
- `AddTag(...)`

### Groups

基本分组：

```go
api := s.Group("/v1")
_ = httpx.GroupGet(api, "/users/{id}", getUser)
_ = httpx.GroupPost(api, "/users", createUser)
```

组级 Huma 能力：

```go
api := s.Group("/admin")
api.UseHumaMiddleware(authMiddleware)
api.DefaultTags("admin")
api.DefaultSecurity(map[string][]string{"bearerAuth": {}})
api.DefaultParameters(&huma.Param{
    Name:   "X-Tenant",
    In:     "header",
    Schema: &huma.Schema{Type: "string"},
})
api.DefaultSummaryPrefix("Admin")
api.DefaultDescription("Administrative APIs")
```

可用组 API：

- `HumaGroup()`
- `UseHumaMiddleware(...)`
- `UseOperationModifier(...)`
- `UseSimpleOperationModifier(...)`
- `UseResponseTransformer(...)`
- `DefaultTags(...)`
- `DefaultSecurity(...)`
- `DefaultParameters(...)`
- `DefaultSummaryPrefix(...)`
- `DefaultDescription(...)`

### 策略路由注册

```go
_ = httpx.RouteWithPolicies(server, httpx.MethodGet, "/resources/{id}", handler,
    httpx.PolicyOperation[GetInput, GetOutput](huma.OperationTags("resources")),
    httpx.PolicyConditionalRead[GetInput, GetOutput](stateGetter),
)
```

可用策略路由 API：

- `RouteWithPolicies(...)`
- `GroupRouteWithPolicies(...)`
- `MustRouteWithPolicies(...)`
- `MustGroupRouteWithPolicies(...)`

### SSE

```go
httpx.MustRouteSSEWithPolicies(server, httpx.MethodGet, "/events", map[string]any{
    "tick": TickEvent{},
    "done": DoneEvent{},
}, func(ctx context.Context, input *StreamInput, send httpx.SSESender) {
    _ = send.Data(TickEvent{Index: 1})
    _ = send(httpx.SSEMessage{ID: 2, Data: DoneEvent{Message: "ok"}})
}, httpx.SSEPolicyOperation[StreamInput](huma.OperationTags("stream")))
```

可用 SSE API：

- `RouteSSEWithPolicies(...)`
- `GroupRouteSSEWithPolicies(...)`
- `MustRouteSSEWithPolicies(...)`
- `MustGroupRouteSSEWithPolicies(...)`
- `SSEPolicyOperation(...)`
- `GetSSE(...)`
- `GroupGetSSE(...)`
- `MustGetSSE(...)`
- `MustGroupGetSSE(...)`

### 条件请求

```go
type GetInput struct {
    httpx.ConditionalParams
}

_ = httpx.RouteWithPolicies(server, httpx.MethodGet, "/resources/{id}", func(ctx context.Context, input *GetInput) (*Output, error) {
    return out, nil
}, httpx.PolicyConditionalRead[GetInput, Output](func(ctx context.Context, input *GetInput) (string, time.Time, error) {
    return currentETag, modifiedAt, nil
}))
```

可用辅助 API：

- `ConditionalParams`
- `PolicyConditionalRead(...)`
- `PolicyConditionalWrite(...)`
- `OperationConditionalRead()`
- `OperationConditionalWrite()`

### Adapter Bridge Hook

```go
httpx.UseAdapter[adapter.LoggerConfigurer](server, func(cfg adapter.LoggerConfigurer) {
    cfg.SetLogger(logger)
})
```

### Graceful Shutdown Hooks（humacli）

```go
cli := humacli.New(func(hooks humacli.Hooks, opts *Options) {
    httpx.BindGracefulShutdownHooks(hooks, server, ":8888")
})
```

## 类型化输入模式

```go
type GetUserInput struct {
    ID int `path:"id"`
}

type ListUsersInput struct {
    Page int `query:"page"`
    Size int `query:"size"`
}

type SecureInput struct {
    RequestID string `header:"X-Request-Id"`
}

type CreateUserInput struct {
    Body struct {
        Name  string `json:"name" validate:"required,min=2,max=64"`
        Email string `json:"email" validate:"required,email"`
    }
}
```

## 中间件模型

`httpx` 使用双层中间件模型：

- 适配器原生中间件：直接在适配器路由器/引擎/应用上注册
- Huma 中间件：通过 `Server.UseHumaMiddleware(...)` 或 `Group.UseHumaMiddleware(...)` 注册

适配器中间件应保持适配器原生：

- `std`: `adapter.Router().Use(...)`
- `gin`: `adapter.Router().Use(...)`
- `echo`: `adapter.Router().Use(...)`
- `fiber`: `adapter.Router().Use(...)`

类型化处理器操作控制在 `httpx` 层：

- `WithPanicRecover(...)` 控制类型化 `httpx` 处理器的 panic 恢复
- `WithAccessLog(...)` 通过服务器日志记录器控制请求日志

运行时监听器设置（如读/写/空闲超时和最大头字节数）是适配器关注点，应在适配器或底层服务器库上配置，而不是通过 `httpx/options.ServerOptions`。

## 日志

`httpx` 日志记录器行为有意在层之间划分：

- `httpx.WithLogger(...)` 配置 `httpx.Server` 日志记录器
- 适配器日志记录器配置 `adapter/std`、`adapter/gin`、`adapter/echo` 和 `adapter/fiber` 发出的桥接层错误
- 框架原生日志记录器和日志记录中间件保持为框架关注点

在实践中这意味着：

- 使用 `httpx.WithLogger(...)` 用于 `httpx` 路由/访问日志/路由注册输出
- 当你希望适配器桥接错误使用相同日志记录器时，显式配置适配器日志记录器
- 继续在适配器路由器或引擎上配置 `chi` / `gin` / `echo` / `fiber` 日志记录中间件

`httpx` 目前不承诺完全替换框架原生日志记录器。

## 适配器构建

监听器和桥接层配置属于适配器，而不是 `httpx.ServerOptions`。

对于基于 `net/http` 的适配器（如 `std`、`gin` 和 `echo`），使用构建时适配器选项：

```go
stdAdapter := std.NewWithOptions(std.Options{
    Logger: slogLogger,
    Server: std.ServerOptions{
        ReadTimeout:     15 * time.Second,
        WriteTimeout:    15 * time.Second,
        IdleTimeout:     60 * time.Second,
        ShutdownTimeout: 5 * time.Second,
        MaxHeaderBytes:  1 << 20,
    },
})
```

对于 `fiber`，超时设置属于适配器创建应用时使用的应用配置：

```go
fiberAdapter := fiber.NewWithOptions(nil, fiber.Options{
    Logger: slogLogger,
    App: fiber.AppOptions{
        ReadTimeout:     15 * time.Second,
        WriteTimeout:    15 * time.Second,
        IdleTimeout:     60 * time.Second,
        ShutdownTimeout: 5 * time.Second,
    },
})
```

如果你传递已创建的框架对象，该框架对象的自己的配置保持权威。

## Introspection API

- `GetRoutes()`
- `GetRoutesByMethod(method)`
- `GetRoutesByPath(prefix)`
- `HasRoute(method, path)`
- `RouteCount()`

## 选项构建器

你可以通过 `httpx/options` 构建服务器选项：

```go
opts := options.DefaultServerOptions()
opts.BasePath = "/api"
opts.HumaTitle = "Arc API"
opts.DocsPath = "/reference"
opts.DocsRenderer = httpx.DocsRendererSwaggerUI
opts.EnablePanicRecover = true
opts.EnableAccessLog = true

s := httpx.NewServer(append(opts.Build(), httpx.WithAdapter(a))...)
```

单独使用适配器构建选项进行监听器超时和适配器日志记录器配置。

## 测试模式

```go
req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
rec := httptest.NewRecorder()
s.ServeHTTP(rec, req)

if rec.Code != http.StatusOK {
    t.Fatal(rec.Code)
}
```

## 常见问题

### 我必须使用 Huma 风格输入 struct 吗？

是的，用于此包中的类型化路由处理器。

### 我仍然可以访问原始 Huma API 吗？

可以。使用 `HumaAPI()`、`OpenAPI()` 和 `HumaGroup()`。

### `httpx` 也应该包装适配器中间件吗？

不。保持适配器原生中间件在适配器本身上，并使用 `httpx` 进行 Huma 端中间件和服务组织。

## 示例

- Quickstart: `go run ./httpx/examples/quickstart`
  - 最小类型化路由 + 验证 + 基础路径
- Auth: `go run ./httpx/examples/auth`
  - 安全方案、全局头和类型化认证头绑定
  - 查看 [`httpx/examples/auth/README.md`](https://github.com/DaiYuANg/arcgo/tree/main/httpx/examples/auth)
- Organization: `go run ./httpx/examples/organization`
  - 文档路径、安全、全局头和组默认值
  - 查看 [`httpx/examples/organization/README.md`](https://github.com/DaiYuANg/arcgo/tree/main/httpx/examples/organization)
- SSE: `go run ./httpx/examples/sse`
  - 基于 `text/event-stream` 的类型化事件流
- Conditional Requests: `go run ./httpx/examples/conditional`
  - 基于 ETag 和 Last-Modified 的前置条件校验
