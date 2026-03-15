---
title: 'ArcGo 文档'
description: '模块化 Go 后端基础设施工具集'
date: '2026-03-08T00:00:00+08:00'
draft: false
---

# ArcGo

**ArcGo** 是一个模块化的 Go 后端基础设施工具集。它按包组织、可按需引入，并允许包间依赖组合。

## 快速开始

```bash
go get github.com/DaiYuANg/archgo/{package}
```

## 核心特性

- 🧩 **模块化组织** - 按包拆分并支持按需引入，允许包间依赖组合（如 `collectionx`、`observabilityx`）
- 🔒 **类型安全** - 基于 Go 泛型与显式接口的强类型 API
- 🧪 **实验性阶段** - 当前处于快速迭代，API 与行为仍可能调整
- 🔗 **依赖可控** - 不锁定单一框架，但会按功能引入必要依赖
- 🔍 **可观测性扩展** - 通过 `observabilityx` 可选对接 OpenTelemetry 与 Prometheus

## 包概览

{{< cards >}}
  {{< card link="/docs/authx" title="authx" subtitle="多场景可扩展的认证与鉴权抽象" icon="lock-closed" >}}
  {{< card link="/docs/collectionx" title="collectionx" subtitle="泛型集合与并发安全结构" icon="collection" >}}
  {{< card link="/docs/configx" title="configx" subtitle="分层配置加载与校验" icon="cog" >}}
  {{< card link="/docs/eventx" title="eventx" subtitle="进程内强类型事件总线" icon="lightning-bolt" >}}
  {{< card link="/docs/httpx" title="httpx" subtitle="多框架统一强类型 HTTP 路由" icon="server" >}}
  {{< card link="/docs/logx" title="logx" subtitle="结构化日志与 slog 互通" icon="document-text" >}}
  {{< card link="/docs/observabilityx" title="observabilityx" subtitle="可选可观测性抽象（OTel/Prometheus）" icon="chart-bar" >}}
{{< /cards >}}

## 典型组合

{{% steps %}}

### API 服务基线

`httpx + configx + logx`

### 事件驱动架构

`eventx + logx`

### 数据密集型工具

`collectionx + configx`

{{% /steps %}}

## 代码示例

### 配置管理

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
)
```

### 事件总线

```go
type UserCreated struct { ID int }
func (e UserCreated) Name() string { return "user.created" }

bus := eventx.New()
eventx.Subscribe(bus, func(ctx context.Context, evt UserCreated) error {
    fmt.Println("User created:", evt.ID)
    return nil
})
bus.Publish(context.Background(), UserCreated{ID: 42})
```

### HTTP 服务

```go
s := httpx.NewServer(
    httpx.WithAdapter(std.New()),
    httpx.WithBasePath("/api"),
)

httpx.Get(s, "/health", func(ctx context.Context, input *struct{}) (*HealthOutput, error) {
    return &HealthOutput{Body: struct{ Status string }{Status: "ok"}}, nil
})
```

## 为什么选择 ArcGo？

{{< callout type="info" icon="information-circle" >}}
  **设计哲学**

  ArcGo 不是重型框架，而是一组精心设计的工具库。每个包都遵循以下原则：

  - **单一职责** - 每个包专注于解决一类问题
  - **接口抽象** - 基于接口而非实现，易于测试和替换
  - **组合优先** - 组件可组合，并可依赖共享基础包（如 `collectionx`/`observabilityx`）
  - **文档优先** - 完整的文档和示例代码
{{< /callout >}}

## 开始使用

选择你需要的包开始：

- 需要容器/数据辅助：从 [collectionx](/docs/collectionx) 开始
- 需要认证/授权：从 [authx](/docs/authx) 开始
- 需要配置管理：从 [configx](/docs/configx) 开始
- 需要事件总线：从 [eventx](/docs/eventx) 开始
- 需要 HTTP 路由：从 [httpx](/docs/httpx) 开始
- 需要日志记录：从 [logx](/docs/logx) 开始
- 需要可观测性：从 [observabilityx](/docs/observabilityx) 开始

## 链接

- [GitHub 仓库](https://github.com/DaiYuANg/archgo)
- [Go 模块](https://pkg.go.dev/github.com/DaiYuANg/archgo)
