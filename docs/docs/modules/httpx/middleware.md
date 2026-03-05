---
sidebar_position: 3
---

# 中间件

`httpx` 的设计理念是直接使用框架原生的中间件生态，而不是提供自己的中间件层。

## 设计理念

**核心设计**：httpx 不强制提供中间件，而是让你直接使用框架原生的中间件生态。

这样做的好处：
- ✅ 享受完整的框架生态系统
- ✅ 无需学习新的中间件 API
- ✅ 可以直接使用框架社区的优秀中间件
- ✅ 降低维护成本

## Gin 中间件

```go
package main

import (
    "github.com/DaiYuANg/arcgo/httpx"
    "github.com/DaiYuANg/arcgo/httpx/adapter/gin"
    "github.com/gin-gonic/gin"
    
    ginCors "github.com/gin-contrib/cors"
    ginJwt "github.com/gin-contrib/jwt"
    ginGzip "github.com/gin-contrib/gzip"
)

func main() {
    ginAdapter := gin.New()
    
    // 使用 Gin 原生中间件
    ginAdapter.Engine().Use(
        gin.Logger(),
        gin.Recovery(),
        
        // CORS
        ginCors.Default(),
        
        // Gzip 压缩
        ginGzip.Gzip(ginGzip.DefaultCompression),
        
        // JWT 认证
        ginJwt.New(ginJwt.Config{
            Secret: "your-secret",
        }),
    )
    
    server := httpx.NewServer(httpx.WithAdapter(ginAdapter))
    // ...
}
```

### 常用 Gin 中间件

```bash
# CORS
go get github.com/gin-contrib/cors

# JWT
go get github.com/gin-contrib/jwt

# Gzip
go get github.com/gin-contrib/gzip

# Pprof
go get github.com/gin-contrib/pprof

# Casbin 权限
go get github.com/gin-contrib/casbin
```

## Fiber 中间件

```go
package main

import (
    "github.com/DaiYuANg/arcgo/httpx"
    "github.com/DaiYuANg/arcgo/httpx/adapter/fiber"
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware"
)

func main() {
    fiberAdapter := fiber.New()
    
    // 使用 Fiber 原生中间件
    fiberAdapter.App().Use(
        // Logger
        middleware.Logger(),
        
        // Recover
        middleware.Recover(),
        
        // CORS
        middleware.CORS.New(),
        
        // Rate Limiting
        middleware.Limiter.New(),
        
        // Helmet (安全头)
        middleware.Helmet.New(),
        
        // Compress
        middleware.Compress(),
        
        // Request ID
        middleware.RequestID(),
    )
    
    server := httpx.NewServer(httpx.WithAdapter(fiberAdapter))
    // ...
}
```

### 常用 Fiber 中间件

```bash
# CORS
go get github.com/gofiber/fiber/v2/middleware/cors

# Rate Limiter
go get github.com/gofiber/fiber/v2/middleware/limiter

# Helmet
go get github.com/gofiber/fiber/v2/middleware/helmet

# Compress
go get github.com/gofiber/fiber/v2/middleware/compress

# JWT
go get github.com/gofiber/jwt/v2

# Casbin
go get github.com/gofiber/casbin
```

## Echo 中间件

```go
package main

import (
    "github.com/DaiYuANg/arcgo/httpx"
    "github.com/DaiYuANg/arcgo/httpx/adapter/echo"
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
)

func main() {
    echoAdapter := echo.New()
    
    // 使用 Echo 原生中间件
    echoAdapter.Engine().Use(
        // Logger
        middleware.Logger(),
        
        // Recover
        middleware.Recover(),
        
        // CORS
        middleware.CORS(),
        
        // Rate Limiting
        middleware.RateLimiter(),
        
        // Secure
        middleware.Secure(),
        
        // Gzip
        middleware.Gzip(),
        
        // Request ID
        middleware.RequestID(),
    )
    
    server := httpx.NewServer(httpx.WithAdapter(echoAdapter))
    // ...
}
```

### 常用 Echo 中间件

```bash
# CORS
go get github.com/labstack/echo/v4/middleware/cors

# Rate Limiter
go get github.com/labstack/echo/v4/middleware/ratelimiter

# JWT
go get github.com/labstack/echo-jwt/v4

# Casbin
go get github.com/labstack/echo-casbin
```

## Chi 中间件

```go
package main

import (
    "github.com/DaiYuANg/arcgo/httpx"
    "github.com/DaiYuANg/arcgo/httpx/adapter/std"
    "github.com/go-chi/chi/v5/middleware"
    
    chiCors "github.com/go-chi/cors"
)

func main() {
    stdAdapter := std.New()
    
    // 使用 Chi 原生中间件
    stdAdapter.Router().Use(
        // Logger
        middleware.Logger,
        
        // Recoverer
        middleware.Recoverer,
        
        // Request ID
        middleware.RequestID,
        
        // Real IP
        middleware.RealIP,
        
        // Timeout
        middleware.Timeout,
        
        // Throttle
        middleware.Throttle(100),
        
        // CORS
        chiCors.Handler(cors.Options{
            AllowedOrigins:   []string{"*"},
            AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
            AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
            ExposedHeaders:   []string{"Link"},
            AllowCredentials: true,
            MaxAge:           300,
        }),
    )
    
    server := httpx.NewServer(httpx.WithAdapter(stdAdapter))
    // ...
}
```

### 常用 Chi 中间件

```bash
# CORS
go get github.com/go-chi/cors

# JWT
go get github.com/go-chi/jwt

# Rate Limiter
go get github.com/go-chi/httprate

# Casbin
go get github.com/casbin/casbin/v2
```

## 自定义中间件

### Gin 自定义中间件

```go
func CustomAuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
            return
        }
        
        // 验证 token
        // ...
        
        c.Next()
    }
}

// 使用
ginAdapter.Engine().Use(CustomAuthMiddleware())
```

### Fiber 自定义中间件

```go
func CustomAuthMiddleware() fiber.Handler {
    return func(c *fiber.Ctx) error {
        token := c.Get("Authorization")
        if token == "" {
            return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
        }
        
        // 验证 token
        // ...
        
        return c.Next()
    }
}

// 使用
fiberAdapter.App().Use(CustomAuthMiddleware())
```

### Echo 自定义中间件

```go
func CustomAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        token := c.Request().Header.Get("Authorization")
        if token == "" {
            return c.JSON(401, map[string]interface{}{"error": "Unauthorized"})
        }
        
        // 验证 token
        // ...
        
        return next(c)
    }
}

// 使用
echoAdapter.Engine().Use(CustomAuthMiddleware())
```

### Chi 自定义中间件

```go
func CustomAuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if token == "" {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        // 验证 token
        // ...
        
        next.ServeHTTP(w, r)
    })
}

// 使用
stdAdapter.Router().Use(CustomAuthMiddleware)
```

## 中间件顺序

中间件的注册顺序很重要，通常按照以下顺序：

```go
// Gin 示例
ginAdapter.Engine().Use(
    // 1. 恢复中间件（最外层）
    gin.Recovery(),
    
    // 2. 日志中间件
    gin.Logger(),
    
    // 3. 安全中间件
    cors.Default(),
    helmet.Default(),
    
    // 4. 限流中间件
    ratelimiter.New(),
    
    // 5. 认证中间件
    jwt.New(),
    
    // 6. 业务中间件
    CustomAuthMiddleware(),
)
```

## 条件中间件

根据环境条件启用中间件：

```go
ginAdapter := gin.New()

// 生产环境启用更多中间件
if os.Getenv("ENV") == "production" {
    ginAdapter.Engine().Use(
        ratelimiter.New(),
        helmet.Default(),
    )
}

// 开发环境启用详细日志
if os.Getenv("ENV") == "development" {
    ginAdapter.Engine().Use(gin.Logger())
}
```
