---
title: 'configx'
linkTitle: 'configx'
description: 'Hierarchical Configuration Loading and Validation'
weight: 3
---

## configx

`configx` is a hierarchical configuration loader built on `koanf` and `validator`.

## Roadmap

- Module roadmap: [configx roadmap](./roadmap)
- Global roadmap: [ArcGo roadmap](../roadmap)

## Supported Features

- `.env` loading (`WithDotenv`)
- Configuration file loading (`WithFiles`)
- Environment variable loading (`WithEnvPrefix`)
- Custom source priority (`WithPriority`)
- Set defaults via map or typed object (`WithDefaults`, `WithDefaultsTyped`, `WithTypedDefaults`)
- Optional validation (`WithValidateLevel`, `WithValidator`)
- Optional observability (`WithObservability`)
- Generic and non-generic loading entry points

## Loading Flow

`configx` merges sources by priority. Later sources override earlier ones.

Default priority:

1. dotenv
2. files
3. environment variables

## Quick Start

```go
type AppConfig struct {
    Name string `validate:"required"`
    Port int    `validate:"required,min=1,max=65535"`
}

cfg, err := configx.LoadTErr[AppConfig](
    configx.WithDotenv(),
    configx.WithFiles("config.yaml"),
    configx.WithEnvPrefix("APP"),
    configx.WithValidateLevel(configx.ValidateLevelStruct),
)
if err != nil {
    panic(err)
}
```

## Common Scenarios

### 1) Local Development (`.env` First)

```go
err := configx.Load(&cfg,
    configx.WithDotenv(".env", ".env.local"),
    configx.WithIgnoreDotenvError(true),
)
```

### 2) File + Environment Variable Override

```go
err := configx.Load(&cfg,
    configx.WithFiles("config.yaml"),
    configx.WithEnvPrefix("APP"),
    configx.WithPriority(configx.SourceFile, configx.SourceEnv),
)
```

### 3) Bootstrap with Defaults Only

```go
err := configx.Load(&cfg,
    configx.WithDefaults(map[string]any{
        "name": "my-service",
        "port": 8080,
    }),
)
```

### 4) Generic Loading API (Recommended)

```go
result := configx.LoadT[AppConfig](
    configx.WithFiles("config.yaml"),
)
if result.IsError() {
    panic(result.Error())
}
cfg := result.MustGet()
```

### 5) Explicit `Config` Object Usage (Dynamic Paths)

```go
c, err := configx.LoadConfig(
    configx.WithFiles("config.yaml"),
)
if err != nil {
    panic(err)
}

name := c.GetString("app.name")
port := c.GetInt("app.port")
exists := c.Exists("app.debug")
all := c.All()
_, _, _, _ = name, port, exists, all
```

### 6) Optional Observability (OTel + Prometheus)

```go
otelObs := otelobs.New()
promObs := promobs.New()
obs := observabilityx.Multi(otelObs, promObs)

err := configx.Load(&cfg,
    configx.WithObservability(obs),
    configx.WithFiles("config.yaml"),
)
```

## Validation Modes

- `ValidateLevelNone`: No validation
- `ValidateLevelStruct`: Run struct validation

If you need custom validators/tags:

```go
v := validator.New(validator.WithRequiredStructEnabled())
err := configx.Load(&cfg,
    configx.WithValidator(v),
    configx.WithValidateLevel(configx.ValidateLevelStruct),
)
```

## Environment Variable Key Mapping

With `WithEnvPrefix("APP")`:

- `APP_DATABASE_HOST` -> `database.host`
- `APP_SERVER_READ_TIMEOUT` -> `server.read.timeout`

## Production Tips

- Keep source priority explicit in production builds.
- Use defaults for non-critical values to reduce startup failures.
- Use validation for critical fields (ports, credentials, hostnames).
- Keep `.env` optional in production unless explicitly required.

## Testing Tips

- Use `WithDefaults` in tests for determinism.
- Avoid real env dependencies in unit tests unless testing isolation of `os.Environ`.
- Use `LoadT[T]` in tests to reduce boilerplate.

## FAQ

### Which source should have highest priority?

In most services, environment variables should be highest priority in production.
Common order is: defaults -> file -> env.

### Should I use `LoadT[T]` or `LoadConfig`?

- Use `LoadT[T]` / `LoadTErr[T]` for strong typing and validation-first workflows.
- Use `LoadConfig` only when you need dynamic getters (`GetString`, `Exists`, `All`) or path-based access.

## Troubleshooting

### Environment variable values not taking effect

Check these first:

- `WithEnvPrefix` matches actual env key prefix.
- `WithPriority` places `SourceEnv` after other sources.
- Env keys map to dot-path format (`APP_DB_HOST` -> `db.host`).

### Validation not running

Validation is disabled by default.
Set `WithValidateLevel(...)`, or wire `WithValidator(...)` plus validation level.

### `.env` file missing causes startup crash

Use `WithIgnoreDotenvError(true)` in environments where `.env` is optional.

### Defaults shape mismatch

Prefer explicit map defaults for dynamic keys.
For strong typing, keep defaults in the target config type and load through `LoadT[T]`/`LoadTErr[T]`.

## Anti-Patterns

- Relying on implicit source priority in production.
- Reading config directly from process env in business code after adopting `configx`.
- Disabling validation for critical fields (ports, credentials, URLs).
- Mixing unrelated prefixes for multiple services in shared environments.

## Examples

- [observability](https://github.com/DaiYuANg/arcgo/tree/main/configx/examples/observability): Load configuration with optional OTel + Prometheus instrumentation.
