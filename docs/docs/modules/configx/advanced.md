---
sidebar_position: 3
---

# 高级用法

本文档介绍 `configx` 的高级用法，包括自定义验证器、配置热重载等。

## 自定义验证器

除了使用内置的验证标签，你还可以注册自定义验证函数：

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
    // 创建自定义 validator
    validate := validator.New()
    
    // 注册自定义验证函数
    validate.RegisterValidation("port_range", func(fl validator.FieldLevel) bool {
        val := fl.Field().String()
        // 验证格式： "1024-65535"
        // ... 自定义验证逻辑
        return true
    })
    
    validate.RegisterValidation("country_code", func(fl validator.FieldLevel) bool {
        val := fl.Field().String()
        // 验证国家代码： "US", "CN", ...
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

## 配置热重载

使用 `LoadConfig` 返回的 `*Config` 对象，可以实现配置热重载：

```go
package main

import (
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"
    "github.com/DaiYuANg/arcgo/configx"
)

type Config struct {
    App struct {
        Name  string `mapstructure:"name"`
        Port  int    `mapstructure:"port"`
    } `mapstructure:"app"`
}

func main() {
    // 加载初始配置
    cfg, err := configx.LoadConfig(
        configx.WithFiles("config.yaml"),
        configx.WithEnvPrefix("APP"),
    )
    if err != nil {
        panic(err)
    }

    // 使用配置
    fmt.Printf("Initial config: port=%d\n", cfg.GetInt("app.port"))

    // 监听信号实现热重载
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGHUP)

    go func() {
        for range sigChan {
            fmt.Println("Received SIGHUP, reloading config...")
            
            // 重新加载配置
            newCfg, err := configx.LoadConfig(
                configx.WithFiles("config.yaml"),
                configx.WithEnvPrefix("APP"),
            )
            if err != nil {
                fmt.Printf("Failed to reload config: %v\n", err)
                continue
            }
            
            cfg = newCfg
            fmt.Printf("Config reloaded: port=%d\n", cfg.GetInt("app.port"))
        }
    }()

    fmt.Println("Server running, send SIGHUP to reload config")
    time.Sleep(time.Hour)
}
```

## 配置合并

可以多次调用 `Load` 来合并配置：

```go
package main

import (
    "github.com/DaiYuANg/arcgo/configx"
)

type Config struct {
    App struct {
        Name  string `mapstructure:"name"`
        Port  int    `mapstructure:"port"`
        Debug bool   `mapstructure:"debug"`
    } `mapstructure:"app"`
}

func main() {
    var cfg Config

    // 先加载默认配置
    err := configx.Load(&cfg,
        configx.WithFiles("config.default.yaml"),
    )
    if err != nil {
        panic(err)
    }

    // 再加载环境特定配置（覆盖默认值）
    err = configx.Load(&cfg,
        configx.WithFiles("config.production.yaml"),
    )
    if err != nil {
        panic(err)
    }

    // 最后加载本地配置（如果存在）
    err = configx.Load(&cfg,
        configx.WithFiles("config.local.yaml"),
    )
    // 忽略文件不存在的错误
    if err != nil && !os.IsNotExist(err) {
        panic(err)
    }
}
```

## 配置路径映射

使用 `mapstructure` 标签可以自定义配置路径映射：

```go
type Config struct {
    // 配置文件中：app_name: my-app
    AppName string `mapstructure:"app_name"`
    
    // 配置文件中：app.name: my-app
    App struct {
        Name string `mapstructure:"name"`
    } `mapstructure:"app"`
    
    // 配置文件中：database 下的所有字段
    Database map[string]interface{} `mapstructure:"database"`
}
```

## 配置类型转换

`configx` 支持自动类型转换：

```go
cfg, _ := configx.LoadConfig(
    configx.WithDefaults(map[string]any{
        "port":      "8080",      // 字符串转 int
        "timeout":   "30s",       // 字符串转 time.Duration
        "debug":     "true",      // 字符串转 bool
        "hosts":     "a,b,c",     // 字符串转 []string
        "ports":     "1,2,3",     // 字符串转 []int
    }),
)

// 自动类型转换
port := cfg.GetInt("port")           // 8080
timeout := cfg.GetDuration("timeout") // 30s
debug := cfg.GetBool("debug")         // true
hosts := cfg.GetStringSlice("hosts")  // ["a", "b", "c"]
ports := cfg.GetIntSlice("ports")     // [1, 2, 3]
```

## 配置子树

可以获取配置的子树：

```go
cfg, _ := configx.LoadConfig(
    configx.WithFiles("config.yaml"),
)

// 获取 database 子树
dbCfg := cfg.Cut("database")

// 现在可以直接获取 database 下的配置
host := dbCfg.GetString("host")
port := dbCfg.GetInt("port")
```

## 配置导出

可以将配置导出为 map 或 JSON：

```go
cfg, _ := configx.LoadConfig(
    configx.WithFiles("config.yaml"),
)

// 导出为 map
allConfig := cfg.All()

// 导出为 JSON
jsonBytes, _ := cfg.MarshalJSON()
fmt.Println(string(jsonBytes))

// 解构到结构体
var config Config
err := cfg.Unmarshal("", &config)
```

## 与 Viper 对比

如果你之前使用 Viper，这里是主要区别：

| 特性 | Viper | configx |
|------|-------|---------|
| 配置源 | 内置 | 基于 koanf，更灵活 |
| 验证 | 需要手动 | 内置 validator 支持 |
| 类型安全 | 弱 | 支持结构体加载 |
| 依赖 | 较少 | 较多（koanf + validator） |

## 最佳实践

### 1. 使用结构体定义配置

```go
// ✅ 推荐
type Config struct {
    App struct {
        Name string `mapstructure:"name" validate:"required"`
        Port int    `mapstructure:"port" validate:"required"`
    } `mapstructure:"app"`
}

// ❌ 不推荐
cfg.GetString("app.name")
cfg.GetInt("app.port")
```

### 2. 设置合理的默认值

```go
configx.Load(&cfg,
    configx.WithDefaults(map[string]any{
        "app.port":  8080,
        "app.debug": false,
    }),
)
```

### 3. 启用验证

```go
configx.Load(&cfg,
    configx.WithValidateLevel(configx.ValidateLevelRequired),
)
```

### 4. 分层加载配置

```go
// 1. 默认值 -> 2. 配置文件 -> 3. 环境变量 -> 4. 命令行参数
configx.Load(&cfg,
    configx.WithDefaults(defaults),
    configx.WithFiles("config.yaml"),
    configx.WithEnvPrefix("APP"),
)
```
