---
sidebar_position: 1
---

# Overview

`configx` is a configuration loader built on [koanf](https://github.com/knadh/koanf) and [validator](https://github.com/go-playground/validator). It supports dotenv + config files + environment variables, customizable priority, and struct validation.

## Features

- `.env` loading
- YAML/JSON/TOML config files
- Environment variables
- Custom source priority
- Default values
- Struct validation via validator
- Clean and practical API

## Install

```bash
go get github.com/DaiYuANg/arcgo/configx
```

## Quick example

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
        configx.WithValidateLevel(configx.ValidateLevelRequired),
    )
    if err != nil {
        panic(err)
    }

    fmt.Printf("Name: %s, Port: %d, Debug: %v\n", cfg.Name, cfg.Port, cfg.Debug)
}
```

## Config sources

### 1. `.env` files

```env
APP_NAME=my-app
APP_PORT=3000
APP_DEBUG=true
DATABASE_HOST=localhost
DATABASE_PORT=5432
```

```go
configx.Load(&cfg,
    configx.WithDotenv(),
    configx.WithDotenv(".env.local", ".env"),
)
```

### 2. Config files

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

```go
configx.Load(&cfg,
    configx.WithFiles("config.yaml"),
)
```

### 3. Environment variables

```go
configx.Load(&cfg,
    configx.WithEnvPrefix("APP"),
)
```

### 4. Defaults

```go
configx.Load(&cfg,
    configx.WithDefaults(map[string]any{
        "app.name":  "my-app",
        "app.port":  8080,
        "app.debug": false,
    }),
)
```

## Priority

Default priority: `.env` < config file < environment variables.

```go
configx.Load(&cfg,
    configx.WithPriority(
        configx.SourceDotenv,
        configx.SourceFile,
        configx.SourceEnv,
    ),
)
```

## Validation levels

| Level | Description |
|------|------|
| `ValidateLevelNone` | Disable validation (default) |
| `ValidateLevelStruct` | Validate struct tags |
| `ValidateLevelRequired` | Validate `required` tags |

## Next

- [Basic Usage](/docs/modules/configx/basic-usage)
- [Advanced](/docs/modules/configx/advanced)
- [API Reference](/docs/modules/configx/api)
