---
sidebar_position: 1
---

# Overview

`httpx` is a flexible adapter layer for popular Go HTTP frameworks.

## Design

Core idea: reduce routing boilerplate while keeping middleware integration native.

- `httpx` handles route registration, endpoint mapping, and OpenAPI integration.
- Middleware should be registered with each framework's native APIs.

## Package layout

```
httpx/
|- adapter/
|  |- gin/
|  |- fiber/
|  |- echo/
|  `- std/
|- examples/
|- huma/
|- middleware/
`- options/
```

## Features

- On-demand adapters by subpackage
- Native middleware support (`Engine()`, `App()`, `Router()`)
- Unified adapter abstraction
- Huma OpenAPI support across adapters
- Internal failure logging via `slog` for JSON/doc rendering failures

## Install

```bash
go get github.com/DaiYuANg/arcgo/httpx/adapter/gin
go get github.com/DaiYuANg/arcgo/httpx/adapter/fiber
go get github.com/DaiYuANg/arcgo/httpx/adapter/echo
go get github.com/DaiYuANg/arcgo/httpx/adapter/std
```

## Quick example (Gin)

```go
package main

import (
    "context"
    "net/http"

    "github.com/DaiYuANg/arcgo/httpx"
    "github.com/DaiYuANg/arcgo/httpx/adapter/gin"
)

type UserEndpoint struct { httpx.BaseEndpoint }

func (e *UserEndpoint) ListUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    e.Success(w, map[string]interface{}{"users": []string{"Alice", "Bob", "Charlie"}})
    return nil
}

func main() {
    ginAdapter := gin.New()
    ginAdapter.Engine().Use(gin.Logger(), gin.Recovery())

    ginAdapter.WithHuma(httpx.ToAdapterHumaOptions(httpx.HumaOptions{
        Enabled: true,
        Title:   "My API",
        Version: "1.0.0",
    }))

    server := httpx.NewServer(
        httpx.WithAdapter(ginAdapter),
        httpx.WithPrintRoutes(true),
    )
    _ = server.Register(&UserEndpoint{})
    server.ListenAndServe(":8080")
}
```

## Why this approach?

1. Avoid dependency conflicts by importing only what you need.
2. Reuse full framework middleware ecosystems.
3. Lower maintenance: no extra middleware layer to maintain.
4. Keep flexibility per project and per framework.

## Dependencies

| Adapter | Dependency |
|------|------|
| `adapter/gin` | `github.com/gin-gonic/gin` |
| `adapter/fiber` | `github.com/gofiber/fiber/v2` |
| `adapter/echo` | `github.com/labstack/echo/v4` |
| `adapter/std` | `github.com/go-chi/chi/v5` |

All adapters also rely on `github.com/danielgtaylor/huma/v2` for OpenAPI generation.

## Next

- [Usage Guide](/docs/modules/httpx/usage)
- [Middleware](/docs/modules/httpx/middleware)
- [Huma OpenAPI](/docs/modules/httpx/huma)
