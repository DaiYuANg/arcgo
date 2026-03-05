---
sidebar_position: 1
---

# 概述

`configx` 是一个基于 [koanf](https://github.com/knadh/koanf) 和 [validator](https://github.com/go-playground/validator) 的配置加载库，支持 dotenv + 配置文件 + 环境变量，可配置优先级，并支持结构体验证。

## 特性

- ✅ **支持 `.env` 文件加载** - 自动加载 `.env` 文件中的环境变量
- ✅ **支持配置文件** - 支持 YAML、JSON、TOML 格式
- ✅ **支持环境变量** - 可直接从环境变量读取配置
- ✅ **可配置加载优先级** - 自定义配置源的加载顺序
- ✅ **支持默认值** - 当配置不存在时使用默认值
- ✅ **结构体验证** - 基于 validator 的结构体标签验证
- ✅ **简洁易用的 API** - 直观的链式调用

## 安装

```bash
go get github.com/DaiYuANg/arcgo/configx
```

## 快速示例

### 基本用法

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
        configx.WithDotenv(),              // 加载 .env 文件
        configx.WithFiles("config.yaml"),  // 加载配置文件
        configx.WithEnvPrefix("APP"),      // 加载 APP_ 开头的环境变量
        configx.WithValidateLevel(configx.ValidateLevelRequired),
    )

    if err != nil {
        panic(err)
    }

    fmt.Printf("Name: %s, Port: %d, Debug: %v\n", cfg.Name, cfg.Port, cfg.Debug)
}
```

### 使用 Config 对象

```go
package main

import (
    "fmt"
    "github.com/DaiYuANg/arcgo/configx"
)

func main() {
    // 加载配置并返回 Config 对象
    cfg, err := configx.LoadConfig(
        configx.WithDotenv(),
        configx.WithFiles("config.yaml"),
        configx.WithEnvPrefix("APP"),
    )
    if err != nil {
        panic(err)
    }

    // 使用 getter 方法
    name := cfg.GetString("app.name")
    port := cfg.GetInt("app.port")
    debug := cfg.GetBool("app.debug")
    timeout := cfg.GetDuration("app.timeout")

    // 解构到结构体
    var config Config
    err = cfg.Unmarshal("", &config)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Name: %s, Port: %d\n", name, port)
}
```

## 配置源

### 1. .env 文件

```env
# .env
APP_NAME=my-app
APP_PORT=3000
APP_DEBUG=true
DATABASE_HOST=localhost
DATABASE_PORT=5432
```

```go
configx.Load(&cfg,
    configx.WithDotenv(), // 默认加载 .env
    configx.WithDotenv(".env.local", ".env"), // 指定多个文件
)
```

### 2. 配置文件

```yaml
# config.yaml
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
    configx.WithFiles("config.json", "config.toml"),
)
```

### 3. 环境变量

```bash
export APP_NAME=my-app
export APP_PORT=3000
```

```go
configx.Load(&cfg,
    configx.WithEnvPrefix("APP"), // 加载 APP_ 开头的环境变量
)
```

### 4. 默认值

```go
configx.Load(&cfg,
    configx.WithDefaults(map[string]any{
        "app.name":  "my-app",
        "app.port":  8080,
        "app.debug": false,
    }),
)
```

## 配置优先级

默认优先级：`.env` < 配置文件 < 环境变量（后者覆盖前者）

### 自定义优先级

```go
// 环境变量优先级最高
configx.Load(&cfg,
    configx.WithPriority(
        configx.SourceDotenv,
        configx.SourceFile,
        configx.SourceEnv,
    ),
)

// 配置文件优先级最高
configx.Load(&cfg,
    configx.WithPriority(
        configx.SourceEnv,
        configx.SourceDotenv,
        configx.SourceFile,
    ),
)
```

## 结构体验证

```go
type Config struct {
    Name     string `mapstructure:"name" validate:"required"`
    Port     int    `mapstructure:"port" validate:"required,min=1024,max=65535"`
    Database struct {
        Host string `mapstructure:"host" validate:"required,hostname"`
        Port int    `mapstructure:"port" validate:"required"`
    } `mapstructure:"database"`
}

// 启用验证
configx.Load(&cfg,
    configx.WithFiles("config.yaml"),
    configx.WithValidateLevel(configx.ValidateLevelRequired),
)
```

### 验证级别

| 级别 | 说明 |
|------|------|
| `ValidateLevelNone` | 不验证（默认） |
| `ValidateLevelStruct` | 验证结构体标签 |
| `ValidateLevelRequired` | 验证 required 标签 |

## 下一步

- [基本用法](/docs/modules/configx/basic-usage) - 学习更多使用示例
- [高级用法](/docs/modules/configx/advanced) - 了解优先级、自定义验证器等
- [API 参考](/docs/modules/configx/api) - 查看完整的 API 文档
