---
title: 'configx'
linkTitle: 'configx'
description: '分层配置加载与校验'
weight: 3
---

## configx

`configx` 是基于 `koanf` 和 `validator` 构建的分层配置加载器。

## 路线图

- 模块路线图：[configx roadmap](./roadmap)
- 全局路线图：[ArcGo roadmap](../roadmap)

## 支持的功能

- `.env` 加载 (`WithDotenv`)
- 配置文件加载 (`WithFiles`)
- 环境变量加载 (`WithEnvPrefix`)
- 自定义源优先级 (`WithPriority`)
- 通过 map 或 struct 设置默认值 (`WithDefaults`, `WithDefaultsTyped`, `WithDefaultsStruct`, `WithDefaultsFrom`)
- 可选验证 (`WithValidateLevel`, `WithValidator`)
- 可选可观测性 (`WithObservability`)
- 泛型和非泛型加载入口点

## 加载流程

`configx` 按优先级合并源。后面的源会覆盖前面的源。

默认优先级：

1. dotenv
2. 文件
3. 环境变量

## 快速开始

```go
type AppConfig struct {
    Name string `mapstructure:"name" validate:"required"`
    Port int    `mapstructure:"port" validate:"required,min=1,max=65535"`
}

var cfg AppConfig
err := configx.Load(&cfg,
    configx.WithDotenv(),
    configx.WithFiles("config.yaml"),
    configx.WithEnvPrefix("APP"),
    configx.WithValidateLevel(configx.ValidateLevelRequired),
)
if err != nil {
    panic(err)
}
```

## 常见场景

### 1) 本地开发（`.env` 优先）

```go
err := configx.Load(&cfg,
    configx.WithDotenv(".env", ".env.local"),
    configx.WithIgnoreDotenvError(true),
)
```

### 2) 文件 + 环境变量覆盖

```go
err := configx.Load(&cfg,
    configx.WithFiles("config.yaml"),
    configx.WithEnvPrefix("APP"),
    configx.WithPriority(configx.SourceFile, configx.SourceEnv),
)
```

### 3) 仅使用默认值引导

```go
err := configx.Load(&cfg,
    configx.WithDefaults(map[string]any{
        "name": "my-service",
        "port": 8080,
    }),
)
```

### 4) 从 struct 设置默认值

```go
type DefaultCfg struct {
    Name string `mapstructure:"name"`
    Port int    `mapstructure:"port"`
}

err := configx.Load(&cfg,
    configx.WithDefaultsStruct(DefaultCfg{Name: "svc", Port: 8080}),
)
```

### 5) 泛型加载 API

```go
result := configx.LoadT[AppConfig](
    configx.WithFiles("config.yaml"),
)
if result.IsError() {
    panic(result.Error())
}
cfg := result.MustGet()
```

### 6) 显式 `Config` 对象使用

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

### 7) 可选可观测性（OTel + Prometheus）

```go
otelObs := otelobs.New()
promObs := promobs.New()
obs := observabilityx.Multi(otelObs, promObs)

err := configx.Load(&cfg,
    configx.WithObservability(obs),
    configx.WithFiles("config.yaml"),
)
```

## 验证模式

- `ValidateLevelNone`: 无验证
- `ValidateLevelStruct`: 运行 struct 验证
- `ValidateLevelRequired`: 强制 required 标签（与 struct 验证路径相同）

如果你需要自定义验证器/标签：

```go
v := validator.New(validator.WithRequiredStructEnabled())
err := configx.Load(&cfg,
    configx.WithValidator(v),
    configx.WithValidateLevel(configx.ValidateLevelRequired),
)
```

## 环境变量键映射

使用 `WithEnvPrefix("APP")`：

- `APP_DATABASE_HOST` -> `database.host`
- `APP_SERVER_READ_TIMEOUT` -> `server.read.timeout`

## 生产环境技巧

- 在生产构建中保持源优先级显式。
- 对非关键值使用默认值以减少启动失败。
- 对关键字段（端口、凭证、主机名）使用验证。
- 除非明确要求，否则在生产环境中保持 `.env` 可选。

## 测试模式

- 在测试中使用 `WithDefaults` 获得确定性。
- 除非测试隔离 `os.Environ`，否则避免在单元测试中使用真实的 env 依赖。
- 在测试中使用 `LoadT[T]` 减少样板代码。

## 常见问题

### 哪个源应该有最高优先级？

在大多数服务中，环境变量在生产环境中应该是最高优先级。
常见顺序是：defaults -> file -> env。

### 我应该使用 `Load` 还是 `LoadConfig`？

- 如果你只需要一个 typed struct，使用 `Load`。
- 如果你还需要在加载后使用动态 getters（`GetString`、`Exists`、`All`），使用 `LoadConfig`。

### Map 默认值 vs struct 默认值？

- `WithDefaults(map[string]any)` 是显式和动态的。
- 当你已经有 typed 默认配置 struct 时，`WithDefaultsStruct` 更方便。

## 故障排除

### 环境变量值不生效

首先检查这些：

- `WithEnvPrefix` 匹配实际 env 键前缀。
- `WithPriority` 将 `SourceEnv` 放在其他源之后。
- Env 键映射到 dot-path 格式（`APP_DB_HOST` -> `db.host`）。

### 验证没有运行

验证默认是禁用的。
设置 `WithValidateLevel(...)`，或 wire `WithValidator(...)` 加上验证级别。

### `.env` 文件缺失导致启动崩溃

在 `.env` 可选的环境中使用 `WithIgnoreDotenvError(true)`。

### `WithDefaultsStruct` 对不支持的类型失败

struct 到 map 的转换是基于反射的。
保持默认 struct 简单，并使用可预测的 `mapstructure` 标签导出字段。

## 反模式

- 在生产环境中依赖隐式源优先级。
- 采用 `configx` 后在业务代码中直接从进程 env 读取配置。
- 对关键字段（端口、凭证、URL）禁用验证。
- 在共享环境中混合多个服务的无关前缀。

## 示例

- [observability](https://github.com/DaiYuANg/archgo/tree/main/configx/examples/observability): 使用可选的 OTel + Prometheus 工具加载配置。
