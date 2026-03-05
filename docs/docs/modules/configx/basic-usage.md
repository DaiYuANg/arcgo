---
sidebar_position: 2
---

# 基本用法

本文档介绍 `configx` 的基本用法和常见场景。

## 加载配置

### 使用 Load 函数

`Load` 函数直接将配置加载到结构体中：

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

### 使用 LoadConfig 函数

`LoadConfig` 返回一个 `*Config` 对象，可以动态获取配置：

```go
package main

import (
    "fmt"
    "github.com/DaiYuANg/arcgo/configx"
)

func main() {
    cfg, err := configx.LoadConfig(
        configx.WithFiles("config.yaml"),
        configx.WithEnvPrefix("APP"),
    )
    if err != nil {
        panic(err)
    }

    // 动态获取配置值
    name := cfg.GetString("app.name")
    port := cfg.GetInt("app.port")
    debug := cfg.GetBool("app.debug")
    
    fmt.Printf("App: %s:%d (debug=%v)\n", name, port, debug)
}
```

## 配置选项

### WithDotenv

加载 `.env` 文件：

```go
// 加载默认的 .env 文件
configx.WithDotenv()

// 加载指定的 .env 文件
configx.WithDotenv(".env.local", ".env.production")
```

### WithFiles

加载配置文件：

```go
// 加载单个文件
configx.WithFiles("config.yaml")

// 加载多个文件（后面的覆盖前面的）
configx.WithFiles("config.default.yaml", "config.yaml", "config.local.yaml")
```

支持的文件格式：
- YAML (`.yaml`, `.yml`)
- JSON (`.json`)
- TOML (`.toml`)

### WithEnvPrefix

设置环境变量前缀：

```go
// 只加载 APP_ 开头的环境变量
configx.WithEnvPrefix("APP")

// 加载多个前缀
configx.WithEnvPrefixs("APP", "CONFIG")
```

### WithDefaults

设置默认值：

```go
configx.WithDefaults(map[string]any{
    "app.name":  "my-app",
    "app.port":  8080,
    "app.debug": false,
    "timeout":   "30s",
})
```

### WithPriority

设置配置源优先级：

```go
// 环境变量优先级最高
configx.WithPriority(
    configx.SourceDotenv,
    configx.SourceFile,
    configx.SourceEnv,
)

// 配置文件优先级最高
configx.WithPriority(
    configx.SourceEnv,
    configx.SourceDotenv,
    configx.SourceFile,
)
```

### WithValidateLevel

设置验证级别：

```go
// 不验证（默认）
configx.WithValidateLevel(configx.ValidateLevelNone)

// 验证结构体标签
configx.WithValidateLevel(configx.ValidateLevelStruct)

// 验证 required 标签
configx.WithValidateLevel(configx.ValidateLevelRequired)
```

## 配置结构体

### 基本结构体

```go
type Config struct {
    Name  string `mapstructure:"name"`
    Port  int    `mapstructure:"port"`
    Debug bool   `mapstructure:"debug"`
}
```

### 嵌套结构体

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

### 带验证标签的结构体

```go
type Config struct {
    Name     string `mapstructure:"name" validate:"required,min=3,max=50"`
    Port     int    `mapstructure:"port" validate:"required,min=1024,max=65535"`
    Email    string `mapstructure:"email" validate:"required,email"`
    Database struct {
        Host string `mapstructure:"host" validate:"required,hostname"`
        Port int    `mapstructure:"port" validate:"required"`
    } `mapstructure:"database"`
}
```

## 配置文件示例

### YAML 配置

```yaml
# config.yaml
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
  ssl: false

logging:
  level: info
  format: json
  outputs:
    - console
    - file
```

### JSON 配置

```json
{
  "app": {
    "name": "my-application",
    "port": 8080,
    "debug": true,
    "timeout": "30s"
  },
  "database": {
    "host": "localhost",
    "port": 5432,
    "user": "admin",
    "password": "secret"
  }
}
```

### TOML 配置

```toml
[app]
name = "my-application"
port = 8080
debug = true
timeout = "30s"

[database]
host = "localhost"
port = 5432
user = "admin"
password = "secret"
```

## .env 文件示例

```env
# 应用配置
APP_NAME=my-app
APP_PORT=3000
APP_DEBUG=true

# 数据库配置
DATABASE_HOST=db.example.com
DATABASE_PORT=5432
DATABASE_USER=admin
DATABASE_PASSWORD=secret

# 日志配置
LOG_LEVEL=info
LOG_FORMAT=json
```

## 完整示例

```go
package main

import (
    "fmt"
    "time"
    "github.com/DaiYuANg/arcgo/configx"
)

type Config struct {
    App struct {
        Name    string        `mapstructure:"name" validate:"required"`
        Port    int           `mapstructure:"port" validate:"required,min=1024,max=65535"`
        Debug   bool          `mapstructure:"debug"`
        Timeout time.Duration `mapstructure:"timeout" validate:"required"`
    } `mapstructure:"app"`
    
    Database struct {
        Host     string `mapstructure:"host" validate:"required,hostname"`
        Port     int    `mapstructure:"port" validate:"required"`
        User     string `mapstructure:"user"`
        Password string `mapstructure:"password"`
        SSL      bool   `mapstructure:"ssl"`
    } `mapstructure:"database"`
    
    Logging struct {
        Level   string   `mapstructure:"level"`
        Format  string   `mapstructure:"format"`
        Outputs []string `mapstructure:"outputs"`
    } `mapstructure:"logging"`
}

func main() {
    var cfg Config

    err := configx.Load(&cfg,
        // 1. 加载默认值
        configx.WithDefaults(map[string]any{
            "app.debug":   false,
            "logging.level": "info",
            "logging.format": "json",
        }),
        // 2. 加载 .env 文件
        configx.WithDotenv(),
        // 3. 加载配置文件
        configx.WithFiles("config.yaml"),
        // 4. 加载环境变量
        configx.WithEnvPrefix("APP"),
        // 5. 启用验证
        configx.WithValidateLevel(configx.ValidateLevelRequired),
    )

    if err != nil {
        panic(err)
    }

    fmt.Printf("App: %s:%d (debug=%v, timeout=%v)\n", 
        cfg.App.Name, cfg.App.Port, cfg.App.Debug, cfg.App.Timeout)
    fmt.Printf("Database: %s:%d (ssl=%v)\n", 
        cfg.Database.Host, cfg.Database.Port, cfg.Database.SSL)
    fmt.Printf("Logging: level=%s, format=%s, outputs=%v\n", 
        cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.Outputs)
}
```
