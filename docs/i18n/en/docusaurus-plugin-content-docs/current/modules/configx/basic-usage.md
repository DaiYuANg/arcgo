---
sidebar_position: 2
---

# Basic Usage

This page covers common `configx` usage patterns.

## Load configuration

### `Load`

`Load` decodes config directly into your struct.

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
}

func main() {
    var cfg Config
    err := configx.Load(&cfg,
        configx.WithFiles("config.yaml"),
    )
    if err != nil {
        panic(err)
    }
    fmt.Printf("Config: %+v\n", cfg)
}
```

### `LoadConfig`

`LoadConfig` returns a `*Config` object for dynamic reads.

```go
cfg, err := configx.LoadConfig(
    configx.WithFiles("config.yaml"),
    configx.WithEnvPrefix("APP"),
)
if err != nil {
    panic(err)
}

name := cfg.GetString("app.name")
port := cfg.GetInt("app.port")
debug := cfg.GetBool("app.debug")
```

## Options

### `WithDotenv`

```go
configx.WithDotenv()
configx.WithDotenv(".env.local", ".env.production")
```

### `WithFiles`

```go
configx.WithFiles("config.yaml")
configx.WithFiles("config.default.yaml", "config.yaml", "config.local.yaml")
```

Supported formats:
- YAML (`.yaml`, `.yml`)
- JSON (`.json`)
- TOML (`.toml`)

### `WithEnvPrefix`

```go
configx.WithEnvPrefix("APP")
configx.WithEnvPrefixs("APP", "CONFIG")
```

### `WithDefaults`

```go
configx.WithDefaults(map[string]any{
    "app.name":  "my-app",
    "app.port":  8080,
    "app.debug": false,
    "timeout":   "30s",
})
```

### `WithPriority`

```go
configx.WithPriority(
    configx.SourceDotenv,
    configx.SourceFile,
    configx.SourceEnv,
)
```

### `WithValidateLevel`

```go
configx.WithValidateLevel(configx.ValidateLevelNone)
configx.WithValidateLevel(configx.ValidateLevelStruct)
configx.WithValidateLevel(configx.ValidateLevelRequired)
```

## Struct mapping

```go
type Config struct {
    App struct {
        Name  string `mapstructure:"name"`
        Port  int    `mapstructure:"port"`
    } `mapstructure:"app"`

    Database struct {
        Host     string `mapstructure:"host"`
        Port     int    `mapstructure:"port"`
        User     string `mapstructure:"user"`
        Password string `mapstructure:"password"`
    } `mapstructure:"database"`
}
```

## Example config files

### YAML

```yaml
app:
  name: my-application
  port: 8080
  debug: true
  timeout: 30s

database:
  host: localhost
  port: 5432
  user: admin
  password: secret
```

### JSON

```json
{
  "app": {
    "name": "my-application",
    "port": 8080,
    "debug": true,
    "timeout": "30s"
  }
}
```

### TOML

```toml
[app]
name = "my-application"
port = 8080
debug = true
timeout = "30s"
```

### .env

```env
APP_NAME=my-app
APP_PORT=3000
APP_DEBUG=true
DATABASE_HOST=db.example.com
DATABASE_PORT=5432
```
