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
- 通过 map 或 typed 对象设置默认值 (`WithDefaults`, `WithDefaultsTyped`, `WithTypedDefaults`)
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

### 4) 泛型加载 API（推荐）

```go
result := configx.LoadT[AppConfig](
    configx.WithFiles("config.yaml"),
)
if result.IsError() {
    panic(result.Error())
}
cfg := result.MustGet()
```

### 5) 显式 `Config` 对象使用（动态路径）

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

### 6) 可选可观测性（OTel + Prometheus）

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

如果你需要自定义验证器/标签：

```go
v := validator.New(validator.WithRequiredStructEnabled())
err := configx.Load(&cfg,
    configx.WithValidator(v),
    configx.WithValidateLevel(configx.ValidateLevelStruct),
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

### 我应该使用 `LoadT[T]` 还是 `LoadConfig`？

- 如果你想要强类型和校验优先，使用 `LoadT[T]` / `LoadTErr[T]`。
- 仅在需要动态 getter（`GetString`、`Exists`、`All`）或路径式读取时使用 `LoadConfig`。

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

### 默认值结构不匹配

动态键请优先使用显式 map 默认值。
如果是强类型配置，请把默认值保留在目标配置类型中，并通过 `LoadT[T]`/`LoadTErr[T]` 加载。

## 反模式

- 在生产环境中依赖隐式源优先级。
- 采用 `configx` 后在业务代码中直接从进程 env 读取配置。
- 对关键字段（端口、凭证、URL）禁用验证。
- 在共享环境中混合多个服务的无关前缀。

## 示例

- [observability](https://github.com/DaiYuANg/arcgo/tree/main/configx/examples/observability): 使用可选的 OTel + Prometheus 工具加载配置。
