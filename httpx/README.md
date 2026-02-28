# HTTPX - 灵活的 HTTP 框架适配器

`httpx` 是一个灵活的 HTTP 框架适配器层，支持多种流行的 Go Web 框架。

## 设计理念

**核心思路：减少样板代码，方便集成和注册 route**

- **httpx 的职责**：统一管理路由注册、端点映射、OpenAPI 文档集成
- **中间件的职责**：直接使用各框架的原生方式注册，享受完整的框架生态

## 包结构

```
httpx/
├── adapter/              # 适配器子包（按需引入）
│   ├── gin/             # Gin 框架适配器
│   ├── fiber/           # Fiber 框架适配器
│   ├── echo/            # Echo 框架适配器
│   └── std/             # 标准库适配器（基于 chi）
├── examples/            # 示例代码
├── huma/               # Huma OpenAPI 集成
├── middleware/         # 通用中间件（可选）
└── options/            # 配置选项
```

## 特性

- **按需引入**：每个适配器都是独立的子包，只引入你需要的框架依赖
- **原生中间件支持**：通过 `Engine()`、`App()`、`Router()` 方法直接使用框架原生的中间件生态
- **统一接口**：所有适配器实现相同的接口，可以无缝切换
- **Huma OpenAPI 支持**：所有适配器都支持 Huma OpenAPI 文档生成

## 安装

根据你需要的框架选择安装：

### 使用 Gin
```bash
go get github.com/DaiYuANg/toolkit4go/httpx/adapter/gin
```

### 使用 Fiber
```bash
go get github.com/DaiYuANg/toolkit4go/httpx/adapter/fiber
```

### 使用 Echo
```bash
go get github.com/DaiYuANg/toolkit4go/httpx/adapter/echo
```

### 使用标准库
```bash
go get github.com/DaiYuANg/toolkit4go/httpx/adapter/std
```

## 使用示例

### Gin 示例

```go
package main

import (
    "context"
    "net/http"

    "github.com/DaiYuANg/toolkit4go/httpx"
    "github.com/DaiYuANg/toolkit4go/httpx/adapter/gin"
    "github.com/gin-gonic/gin"
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
    // 1. 创建 Gin 适配器
    ginAdapter := gin.New()
    
    // 2. 【重要】使用 Gin 原生方式注册中间件
    // 你可以使用任何 Gin 生态的中间件
    ginAdapter.Engine().Use(
        gin.Logger(),
        gin.Recovery(),
        cors.Default(), // github.com/gin-contrib/cors
    )
    
    // 3. 启用 Huma OpenAPI 文档（可选）
    ginAdapter.WithHuma(httpx.ToAdapterHumaOptions(httpx.HumaOptions{
        Enabled:     true,
        Title:       "My API",
        Version:     "1.0.0",
    }))

    // 4. 创建服务器并注册路由
    server := httpx.NewServer(
        httpx.WithAdapter(ginAdapter),
        httpx.WithPrintRoutes(true),
    )
    _ = server.Register(&UserEndpoint{})
    server.ListenAndServe(":8080")
}
```

### Fiber 示例

```go
package main

import (
    "context"
    "net/http"

    "github.com/DaiYuANg/toolkit4go/httpx"
    "github.com/DaiYuANg/toolkit4go/httpx/adapter/fiber"
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
    // 1. 创建 Fiber 适配器
    fiberAdapter := fiber.New()
    
    // 2. 【重要】使用 Fiber 原生方式注册中间件
    fiberAdapter.App().Use(
        // fiber.Logger(),
        // fiber.Recover(),
        // cors.New(), // github.com/gofiber/fiber/v2/middleware/cors
    )
    
    // 3. 启用 Huma OpenAPI 文档
    fiberAdapter.WithHuma(httpx.ToAdapterHumaOptions(httpx.HumaOptions{
        Enabled:     true,
        Title:       "My API",
        Version:     "1.0.0",
    }))

    // 4. 创建服务器并注册路由
    server := httpx.NewServer(
        httpx.WithAdapter(fiberAdapter),
    )
    _ = server.Register(&UserEndpoint{})
    server.ListenAndServe(":8080")
}
```

### Echo 示例

```go
package main

import (
    "context"
    "net/http"

    "github.com/DaiYuANg/toolkit4go/httpx"
    "github.com/DaiYuANg/toolkit4go/httpx/adapter/echo"
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
    // 1. 创建 Echo 适配器
    echoAdapter := echo.New()
    
    // 2. 【重要】使用 Echo 原生方式注册中间件
    echoAdapter.Engine().Use(
        // echo.Logger(),
        // echo.Recover(),
        // middleware.CORS(), // github.com/labstack/echo/v4/middleware
    )
    
    // 3. 启用 Huma OpenAPI 文档
    echoAdapter.WithHuma(httpx.ToAdapterHumaOptions(httpx.HumaOptions{
        Enabled:     true,
        Title:       "My API",
        Version:     "1.0.0",
    }))

    // 4. 创建服务器并注册路由
    server := httpx.NewServer(
        httpx.WithAdapter(echoAdapter),
    )
    _ = server.Register(&UserEndpoint{})
    server.ListenAndServe(":8080")
}
```

### 标准库示例

```go
package main

import (
    "context"
    "net/http"

    "github.com/DaiYuANg/toolkit4go/httpx"
    "github.com/DaiYuANg/toolkit4go/httpx/adapter/std"
    "github.com/go-chi/chi/v5/middleware"
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
    // 1. 创建 std 适配器（基于 chi）
    stdAdapter := std.New()
    
    // 2. 【重要】使用 chi 原生方式注册中间件
    stdAdapter.Router().Use(
        middleware.Logger,
        middleware.Recoverer,
        middleware.RequestID,
        middleware.CORS(), // github.com/go-chi/cors
    )
    
    // 3. 启用 Huma OpenAPI 文档
    stdAdapter.WithHuma(httpx.ToAdapterHumaOptions(httpx.HumaOptions{
        Enabled:     true,
        Title:       "My API",
        Version:     "1.0.0",
    }))

    // 4. 创建服务器并注册路由
    server := httpx.NewServer(
        httpx.WithAdapter(stdAdapter),
    )
    _ = server.Register(&UserEndpoint{})
    server.ListenAndServe(":8080")
}
```

## 中间件使用

**核心设计**：httpx 不强制提供中间件，而是让你直接使用框架原生的中间件生态。

### Gin 中间件

```go
ginAdapter := gin.New()
ginAdapter.Engine().Use(
    gin.Logger(),
    gin.Recovery(),
    cors.Default(),              // github.com/gin-contrib/cors
    jwt.New(...),                // github.com/gin-contrib/jwt
    gzip.Gzip(gzip.DefaultCompression), // github.com/gin-contrib/gzip
)
```

### Fiber 中间件

```go
fiberAdapter := fiber.New()
fiberAdapter.App().Use(
    fiber.Logger(),
    fiber.Recover(),
    cors.New(),                  // github.com/gofiber/fiber/v2/middleware/cors
    limiter.New(),               // github.com/gofiber/fiber/v2/middleware/limiter
    helmet.New(),                // github.com/gofiber/helmet
)
```

### Echo 中间件

```go
echoAdapter := echo.New()
echoAdapter.Engine().Use(
    echo.Logger(),
    echo.Recover(),
    middleware.CORS(),           // github.com/labstack/echo/v4/middleware
    middleware.RateLimiter(),    // github.com/labstack/echo/v4/middleware
    middleware.Secure(),
)
```

### Chi 中间件

```go
stdAdapter := std.New()
stdAdapter.Router().Use(
    middleware.Logger,
    middleware.Recoverer,
    middleware.RequestID,
    middleware.Timeout(),
    cors.Handler(cors.Options{}), // github.com/go-chi/cors
)
```

## 配置选项

使用 `options` 包进行统一配置：

```go
import "github.com/DaiYuANg/toolkit4go/httpx/options"

serverOpts := options.DefaultServerOptions()
serverOpts.Logger = slogLogger
serverOpts.BasePath = "/api"
serverOpts.PrintRoutes = true
serverOpts.HumaEnabled = true
serverOpts.HumaTitle = "My API"

// 创建适配器并注册中间件
stdAdapter := std.New()
stdAdapter.Router().Use(middleware.Logger, middleware.Recoverer)

server := httpx.NewServer(append(serverOpts.Build(), httpx.WithAdapter(stdAdapter))...)
```

## 依赖说明

- `adapter/gin` - 依赖 `github.com/gin-gonic/gin`
- `adapter/fiber` - 依赖 `github.com/gofiber/fiber/v2`
- `adapter/echo` - 依赖 `github.com/labstack/echo/v4`
- `adapter/std` - 依赖 `github.com/go-chi/chi/v5`

所有适配器都依赖 `github.com/danielgtaylor/huma/v2` 用于 OpenAPI 文档生成。

## 为什么这样设计？

1. **减少依赖冲突**：每个适配器独立子包，只引入需要的框架
2. **完整生态支持**：直接使用框架原生中间件，享受完整的生态系统
3. **降低维护成本**：不需要为每个框架维护一套中间件
4. **灵活性**：用户可以根据需求自由选择中间件组合

## License

MIT License
