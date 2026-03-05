---
sidebar_position: 3
---

# Advanced Usage

This page covers advanced `configx` patterns, including custom validators and reload strategies.

## Custom validator

```go
package main

import (
    "github.com/DaiYuANg/arcgo/configx"
    "github.com/go-playground/validator/v10"
)

type Config struct {
    PortRange string `mapstructure:"port_range" validate:"port_range"`
    Country   string `mapstructure:"country" validate:"country_code"`
}

func main() {
    validate := validator.New()

    validate.RegisterValidation("port_range", func(fl validator.FieldLevel) bool {
        val := fl.Field().String()
        _ = val
        return true
    })

    validate.RegisterValidation("country_code", func(fl validator.FieldLevel) bool {
        val := fl.Field().String()
        return len(val) == 2
    })

    var cfg Config
    err := configx.Load(&cfg,
        configx.WithFiles("config.yaml"),
        configx.WithValidator(validate),
    )
    if err != nil {
        panic(err)
    }
}
```

## Hot reload strategy

Use `LoadConfig` and reload on signal (or your own trigger):

```go
cfg, err := configx.LoadConfig(
    configx.WithFiles("config.yaml"),
    configx.WithEnvPrefix("APP"),
)
if err != nil {
    panic(err)
}

// Reload when needed
newCfg, err := configx.LoadConfig(
    configx.WithFiles("config.yaml"),
    configx.WithEnvPrefix("APP"),
)
if err == nil {
    cfg = newCfg
}
```

## Merge layered configs

```go
var cfg Config

_ = configx.Load(&cfg, configx.WithFiles("config.default.yaml"))
_ = configx.Load(&cfg, configx.WithFiles("config.production.yaml"))
_ = configx.Load(&cfg, configx.WithFiles("config.local.yaml"))
```

## Path mapping

```go
type Config struct {
    AppName string `mapstructure:"app_name"`

    App struct {
        Name string `mapstructure:"name"`
    } `mapstructure:"app"`

    Database map[string]interface{} `mapstructure:"database"`
}
```

## Type conversion

```go
cfg, _ := configx.LoadConfig(
    configx.WithDefaults(map[string]any{
        "port":    "8080",
        "timeout": "30s",
        "debug":   "true",
        "hosts":   "a,b,c",
        "ports":   "1,2,3",
    }),
)

port := cfg.GetInt("port")
timeout := cfg.GetDuration("timeout")
debug := cfg.GetBool("debug")
hosts := cfg.GetStringSlice("hosts")
ports := cfg.GetIntSlice("ports")
```

## Sub-tree access

```go
dbCfg := cfg.Cut("database")
host := dbCfg.GetString("host")
port := dbCfg.GetInt("port")
```

## Export config

```go
allConfig := cfg.All()
jsonBytes, _ := cfg.MarshalJSON()

var out Config
_ = cfg.Unmarshal("", &out)
```

## Comparison with Viper

| Feature | Viper | configx |
|------|------|------|
| Source model | Built-in | koanf-based, flexible |
| Validation | Manual | validator integration |
| Type safety | Weak | Struct loading |
| Dependencies | Fewer | More (`koanf + validator`) |

## Best practices

1. Use structs as your config schema.
2. Always define safe defaults.
3. Enable validation in non-trivial services.
4. Load config in layers: defaults -> files -> environment.
