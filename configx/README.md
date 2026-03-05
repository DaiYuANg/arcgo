# configx

基于 [koanf](https://github.com/knadh/koanf) 和 [validator](https://github.com/go-playground/validator) 的配置加载库，支持 dotenv + 配置文件 + 环境变量，可配置优先级，并支持结构体验证。

## 特性

- ✅ 支持 `.env` 文件加载
- ✅ 支持配置文件 (YAML/JSON/TOML)
- ✅ 支持环境变量
- ✅ 可配置加载优先级
- ✅ 支持默认值
- ✅ 基于 validator 的结构体验证
- ✅ 简洁易用的 API

## 安装

```bash
go get github.com/DaiYuANg/arcgo/configx
```

## 快速开始

### 基本用法

```go
package main

import (
    "fmt"
    "github.com/DaiYuANg/arcgo/configx"
)

type Config struct {
    Name string `mapstructure:"name" validate:"required"`
    Port int    `mapstructure:"port" validate:"required,min=1024,max=65535"`
    Debug bool  `mapstructure:"debug"`
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

### 配置优先级

默认优先级：`.env` < 配置文件 < 环境变量（后者覆盖前者）

自定义优先级：

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

### 设置默认值

```go
configx.Load(&cfg,
    configx.WithDefaults(map[string]any{
        "app.name": "my-app",
        "app.port": 8080,
        "app.debug": false,
    }),
)
```

### 结构体验证

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

- `ValidateLevelNone` - 不验证（默认）
- `ValidateLevelStruct` - 验证结构体标签
- `ValidateLevelRequired` - 验证 required 标签

## API 参考

### 选项函数

| 函数 | 说明 |
|------|------|
| `WithDotenv(files ...string)` | 启用 .env 文件加载 |
| `WithEnvPrefix(prefix string)` | 设置环境变量前缀 |
| `WithFiles(files ...string)` | 设置配置文件路径 |
| `WithPriority(p ...Source)` | 设置配置源优先级 |
| `WithDefaults(m map[string]any)` | 设置默认值 |
| `WithValidateLevel(level ValidateLevel)` | 设置验证级别 |
| `WithValidator(v *validator.Validate)` | 设置自定义 validator |

### Config 方法

| 方法 | 说明 |
|------|------|
| `GetString(path string) string` | 获取字符串 |
| `GetInt(path string) int` | 获取整数 |
| `GetInt64(path string) int64` | 获取 64 位整数 |
| `GetFloat64(path string) float64` | 获取浮点数 |
| `GetBool(path string) bool` | 获取布尔值 |
| `GetDuration(path string) time.Duration` | 获取时长 |
| `GetStringSlice(path string) []string` | 获取字符串切片 |
| `GetIntSlice(path string) []int` | 获取整数切片 |
| `Unmarshal(path string, out any) error` | 解构到结构体 |
| `Exists(path string) bool` | 检查键是否存在 |
| `All() map[string]any` | 获取所有配置 |

## 示例配置文件

### config.yaml

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

### .env

```env
APP_NAME=my-app
APP_PORT=3000
DATABASE_HOST=db.example.com
DATABASE_PORT=5432
```

## License

MIT
