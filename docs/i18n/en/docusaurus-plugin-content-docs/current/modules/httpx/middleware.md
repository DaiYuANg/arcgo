---
sidebar_position: 3
---

# Middleware

`httpx` intentionally relies on framework-native middleware instead of introducing another middleware abstraction.

## Why

- Keep full framework ecosystem compatibility
- No extra middleware API to learn
- Reuse battle-tested community middleware
- Lower maintenance cost

## Gin

```go
ginAdapter := gin.New()
ginAdapter.Engine().Use(
    gin.Logger(),
    gin.Recovery(),
    ginCors.Default(),
    ginGzip.Gzip(ginGzip.DefaultCompression),
)
```

## Fiber

```go
fiberAdapter := fiber.New()
fiberAdapter.App().Use(
    middleware.Logger(),
    middleware.Recover(),
    cors.New(),
    limiter.New(),
    helmet.New(),
)
```

## Echo

```go
echoAdapter := echo.New()
echoAdapter.Engine().Use(
    middleware.Logger(),
    middleware.Recover(),
    middleware.CORS(),
    middleware.RequestID(),
)
```

## Chi (`adapter/std`)

```go
stdAdapter := std.New()
stdAdapter.Router().Use(
    middleware.Logger,
    middleware.Recoverer,
    middleware.RequestID,
)
```

## Custom middleware examples

### Gin

```go
func CustomAuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        if c.GetHeader("Authorization") == "" {
            c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
            return
        }
        c.Next()
    }
}
```

### Fiber

```go
func CustomAuthMiddleware() fiber.Handler {
    return func(c *fiber.Ctx) error {
        if c.Get("Authorization") == "" {
            return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
        }
        return c.Next()
    }
}
```

### Echo

```go
func CustomAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        if c.Request().Header.Get("Authorization") == "" {
            return c.JSON(401, map[string]interface{}{"error": "Unauthorized"})
        }
        return next(c)
    }
}
```

### Chi

```go
func CustomAuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Header.Get("Authorization") == "" {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

## Recommended order

1. Recovery
2. Logging
3. Security (CORS/headers)
4. Rate limiting
5. Authentication/authorization
6. Business-specific middleware
