---
title: 'configx 配置源与优先级'
linkTitle: 'sources-priority'
description: '从文件、环境变量与命令行参数加载，并控制合并顺序'
weight: 3
---

## 配置源与优先级

后加载的源会**覆盖**先加载的源。默认顺序为 **dotenv → file → env → args**。

本页示例使用**临时 YAML 文件**、`os.Setenv` 与 `pflag`，确保可自包含复制运行。

## 1）从 YAML 文件加载

```go
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/DaiYuANg/arcgo/configx"
)

type AppConfig struct {
	Name string `validate:"required"`
	Port int    `validate:"required,min=1,max=65535"`
}

func main() {
	dir, err := os.MkdirTemp("", "configx-doc-*")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("name: from-yaml\nport: 3000\n"), 0o644); err != nil {
		log.Fatal(err)
	}

	cfg, err := configx.LoadTErr[AppConfig](
		configx.WithFiles(path),
		configx.WithValidateLevel(configx.ValidateLevelStruct),
	)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%+v", cfg)
}
```

## 2）环境变量覆盖文件值

使用 `WithEnvPrefix("APP")` 时，类似 `APP_PORT` 会映射到 `port`（默认分隔符 `_` 会映射为 `.` 层级路径）。

```go
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/DaiYuANg/arcgo/configx"
)

type AppConfig struct {
	Name string `validate:"required"`
	Port int    `validate:"required,min=1,max=65535"`
}

func main() {
	dir, err := os.MkdirTemp("", "configx-doc-*")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("name: from-yaml\nport: 3000\n"), 0o644); err != nil {
		log.Fatal(err)
	}

	if err := os.Setenv("APP_PORT", "4000"); err != nil {
		log.Fatal(err)
	}
	defer func() { _ = os.Unsetenv("APP_PORT") }()

	cfg, err := configx.LoadTErr[AppConfig](
		configx.WithFiles(path),
		configx.WithEnvPrefix("APP"),
		configx.WithValidateLevel(configx.ValidateLevelStruct),
	)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%+v", cfg)
}
```

## 3）显式指定 `WithPriority`

当你只关心 **file** 与 **env** 时，可以显式写出合并顺序（env 放后面即优先级更高）。

```go
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/DaiYuANg/arcgo/configx"
)

type AppConfig struct {
	Name string `validate:"required"`
	Port int    `validate:"required,min=1,max=65535"`
}

func main() {
	dir, err := os.MkdirTemp("", "configx-doc-*")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("name: from-yaml\nport: 3000\n"), 0o644); err != nil {
		log.Fatal(err)
	}

	if err := os.Setenv("APP_PORT", "5000"); err != nil {
		log.Fatal(err)
	}
	defer func() { _ = os.Unsetenv("APP_PORT") }()

	cfg, err := configx.LoadTErr[AppConfig](
		configx.WithFiles(path),
		configx.WithEnvPrefix("APP"),
		configx.WithPriority(configx.SourceFile, configx.SourceEnv),
		configx.WithValidateLevel(configx.ValidateLevelStruct),
	)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%+v", cfg)
}
```

## 4）命令行参数覆盖环境变量与文件值

`SourceArgs` 现在同时支持两种入口：

- `WithArgs(...)` / `WithOSArgs()`：直接读取原始长参数
- `WithFlagSet(fs)` / `WithCommandLineFlags()`：读取 `pflag` 中被显式设置过的 flag

下面的例子里，文件中的 `3000` 和环境变量里的 `4000` 最终都会被命令行里的 `6000` 覆盖。

```go
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/DaiYuANg/arcgo/configx"
)

type AppConfig struct {
	Name string `validate:"required"`
	Port int    `validate:"required,min=1,max=65535"`
}

func main() {
	dir, err := os.MkdirTemp("", "configx-doc-*")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("name: from-yaml\nport: 3000\n"), 0o644); err != nil {
		log.Fatal(err)
	}

	if err := os.Setenv("APP_PORT", "4000"); err != nil {
		log.Fatal(err)
	}
	defer func() { _ = os.Unsetenv("APP_PORT") }()

	cfg, err := configx.LoadTErr[AppConfig](
		configx.WithFiles(path),
		configx.WithEnvPrefix("APP"),
		configx.WithArgs("--name=from-cli", "--port", "6000"),
		configx.WithValidateLevel(configx.ValidateLevelStruct),
	)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%+v", cfg)
}
```

原始参数支持：

- `--key=value`
- `--key value`
- `--flag`
- `--no-flag`

另外，位置参数会被忽略；遇到单独的 `--` 后停止解析。

## 5）接入 `pflag.FlagSet`

如果你的程序已经在用 `pflag`，直接把解析后的 `FlagSet` 交给 `configx` 即可。

```go
package main

import (
	"log"

	"github.com/DaiYuANg/arcgo/configx"
	"github.com/spf13/pflag"
)

type AppConfig struct {
	Server struct {
		Port int `validate:"required,min=1,max=65535"`
	}
	Debug bool
}

func main() {
	fs := pflag.NewFlagSet("app", pflag.ContinueOnError)
	fs.Int("server-port", 0, "")
	fs.Bool("debug", false, "")

	if err := fs.Parse([]string{"--server-port=7000", "--debug"}); err != nil {
		log.Fatal(err)
	}

	cfg, err := configx.LoadTErr[AppConfig](
		configx.WithFlagSet(fs),
		configx.WithValidateLevel(configx.ValidateLevelStruct),
	)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%+v", cfg)
}
```

`WithFlagSet` 只会读取**显式设置过**的 flag，`pflag` 的默认值不会回灌并覆盖 file/env/defaults。

## 环境变量与命令行键映射

使用 `WithEnvPrefix("APP")` 且默认分隔符为 `_` 时：

- `APP_PORT` → `port`
- `APP_DATABASE_HOST` → `database.host`

使用默认的 `WithArgsNameFunc` 时：

- `--server-port` → `server.port`
- `--db-read-timeout` → `db.read.timeout`
- `--no-debug` → `debug = false`

如果你不想用 `-` 到 `.` 的默认映射，可以通过 `WithArgsNameFunc` 自定义。

## 延伸阅读

- [快速开始](./getting-started)
- [校验与动态访问](./validation-and-dynamic)
