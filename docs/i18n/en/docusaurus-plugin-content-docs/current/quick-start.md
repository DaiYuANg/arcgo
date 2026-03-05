---
sidebar_position: 2
---

# Quick Start

This guide helps you get started with arcgo quickly.

## Requirements

- Go 1.25.0 or later
- Node.js 20.0+ (only required to run the docs site)

## Installation

### Install specific Go modules

```bash
# Configuration loader
go get github.com/DaiYuANg/arcgo/configx

# Logger
go get github.com/DaiYuANg/arcgo/logx

# HTTP adapters (choose one)
go get github.com/DaiYuANg/arcgo/httpx/adapter/gin
go get github.com/DaiYuANg/arcgo/httpx/adapter/fiber
go get github.com/DaiYuANg/arcgo/httpx/adapter/echo
go get github.com/DaiYuANg/arcgo/httpx/adapter/std
```

### Install all modules

```bash
go get github.com/DaiYuANg/arcgo/...
```

## Quick examples

### 1. Load config with configx

`config.yaml`:

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

`.env`:

```env
APP_NAME=my-app
APP_PORT=3000
DATABASE_HOST=db.example.com
```

Go code:

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

### 2. Log with logx

```go
package main

import (
    "fmt"
    "github.com/DaiYuANg/arcgo/logx"
)

func main() {
    logger := logx.MustNew(
        logx.WithConsole(true),
        logx.WithFile("/var/log/app.log"),
        logx.WithLevel(logx.InfoLevel),
        logx.WithCaller(true),
    )
    defer logger.Close()

    logger.Info("Server started", "port", 8080)
    logger.Debug("Debug info", "key", "value")

    requestLogger := logger.WithField("request_id", "12345")
    requestLogger.Info("Processing request")

    if err := someFunction(); err != nil {
        logger.WithError(err).Error("Operation failed")
    }
}

func someFunction() error {
    return fmt.Errorf("sample error")
}
```

### 3. Build an HTTP service with httpx

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

func (e *UserEndpoint) GetUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    id := e.Param(w, r, "id")
    e.Success(w, map[string]interface{}{
        "id":   id,
        "name": "User " + id,
    })
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
        httpx.WithBasePath("/api"),
        httpx.WithPrintRoutes(true),
    )

    _ = server.Register(&UserEndpoint{})
    server.ListenAndServe(":8080")
}
```

Visit `http://localhost:8080/docs` for OpenAPI docs.

## Next steps

- [configx overview](/docs/modules/configx/overview)
- [httpx overview](/docs/modules/httpx/overview)
- [logx overview](/docs/modules/logx/overview)
