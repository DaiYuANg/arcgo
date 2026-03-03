---
sidebar_position: 1
---

# Welcome to toolkit4go

**toolkit4go** is a concise and efficient Go toolkit focused on common infrastructure components, helping you reduce boilerplate and focus on business logic.

## Design goals

- **Simple API**: easy to understand and use
- **Flexible and extensible**: modular design with on-demand imports
- **Production-ready**: practical and stable
- **Ecosystem-friendly**: integrates with mainstream Go libraries

## Core modules

### 1. configx - Configuration loading

Built on [koanf](https://github.com/knadh/koanf), with support for multiple config sources and struct validation.

Key features:
- `.env`, config files (YAML/JSON/TOML), and environment variables
- Customizable source priority
- Default values
- Validator-based struct validation

```go
package main

import (
    "fmt"
    "github.com/DaiYuANg/toolkit4go/configx"
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

[View docs ->](/docs/modules/configx/overview)

---

### 2. httpx - HTTP framework adapters

A unified adapter layer for Gin, Fiber, Echo, and Chi-based standard net/http setup.

Key features:
- Adapter per framework (independent subpackage)
- Native middleware usage
- Shared adapter interface
- Huma OpenAPI integration

```go
package main

import (
    "context"
    "net/http"

    "github.com/DaiYuANg/toolkit4go/httpx"
    "github.com/DaiYuANg/toolkit4go/httpx/adapter/gin"
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

[View docs ->](/docs/modules/httpx/overview)

---

### 3. logx - Logger

A high-performance logger based on zerolog, with file rotation and error tracking support.

Key features:
- Console/file output
- Rotation via lumberjack
- oops-based stack tracing
- Development/production presets

```go
package main

import "github.com/DaiYuANg/toolkit4go/logx"

func main() {
    logger := logx.MustNew(
        logx.WithConsole(true),
        logx.WithFile("/var/log/app.log"),
        logx.WithLevel(logx.InfoLevel),
    )
    defer logger.Close()

    logger.Info("Server started", "port", 8080)
}
```

[View docs ->](/docs/modules/logx/overview)

---

## Get started

Continue with [Quick Start](/docs/quick-start).
