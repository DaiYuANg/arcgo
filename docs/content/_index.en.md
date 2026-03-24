---
title: 'ArcGo Documentation'
description: 'Modular Go Backend Infrastructure Toolkit'
date: '2026-03-08T00:00:00+08:00'
draft: false
---

# ArcGo

**ArcGo** is a modular Go backend infrastructure toolkit. It is package-oriented, supports incremental adoption, and allows inter-package composition.

## Quick Start

```bash
go get github.com/DaiYuANg/arcgo/{package}
```

## Core Features

- **Modular Organization** - Split by package, adopt incrementally, and compose with inter-package dependencies (for example `collectionx`, `observabilityx`)
- **Type Safety** - Strongly typed APIs built with Go generics and explicit interfaces
- **Experimental Stage** - The project is under active iteration; APIs and behavior may still change
- **Dependency-Transparent** - Not locked to one framework, but introduces required dependencies per feature
- **Observability Extensions** - Optional OpenTelemetry and Prometheus integration via `observabilityx`

## Package Overview

{{< cards >}}
  {{< card link="/docs/authx" title="authx" subtitle="Extensible authentication and authorization abstraction" icon="lock-closed" >}}
  {{< card link="/docs/collectionx" title="collectionx" subtitle="Generic collections and concurrency-safe structures" icon="collection" >}}
  {{< card link="/docs/configx" title="configx" subtitle="Hierarchical configuration loading and validation" icon="cog" >}}
  {{< card link="/docs/dix" title="dix" subtitle="Strongly typed modular app framework built on do" icon="cube" >}}
  {{< card link="/docs/eventx" title="eventx" subtitle="In-process strongly typed event bus" icon="lightning-bolt" >}}
  {{< card link="/docs/httpx" title="httpx" subtitle="Multi-framework unified strongly typed HTTP routing" icon="server" >}}
  {{< card link="/docs/logx" title="logx" subtitle="Structured logging with slog interoperability" icon="document-text" >}}
  {{< card link="/docs/observabilityx" title="observabilityx" subtitle="Optional observability abstraction (OTel/Prometheus)" icon="chart-bar" >}}
{{< /cards >}}

## Typical Combinations

{{% steps %}}

### API Service Baseline

`httpx + configx + logx`

### Modular App Baseline

`dix + configx + logx`

### Event-Driven Architecture

`eventx + logx`

### Data-Intensive Tools

`collectionx + configx`

{{% /steps %}}

## Code Examples

### Configuration Management

```go
type AppConfig struct {
    Name string `validate:"required"`
    Port int    `validate:"required,min=1,max=65535"`
}

cfg, err := configx.LoadTErr[AppConfig](
    configx.WithDotenv(),
    configx.WithFiles("config.yaml"),
    configx.WithEnvPrefix("APP"),
)
```

### Event Bus

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

### HTTP Service

```go
s := httpx.NewServer(
    httpx.WithAdapter(std.New()),
    httpx.WithBasePath("/api"),
)

httpx.Get(s, "/health", func(ctx context.Context, input *struct{}) (*HealthOutput, error) {
    return &HealthOutput{Body: struct{ Status string }{Status: "ok"}}, nil
})
```

## Why Choose ArcGo?

{{< callout type="info" icon="information-circle" >}}
  **Design Philosophy**
  
  ArcGo is not a heavy framework, but a set of carefully designed utility libraries. Each package follows these principles:
  
  - **Single Responsibility** - Each package focuses on solving one type of problem
  - **Interface Abstraction** - Based on interfaces rather than implementations, easy to test and replace
  - **Composition First** - Components are composable and may rely on shared base packages (for example `collectionx`/`observabilityx`)
  - **Documentation First** - Complete documentation and example code
{{< /callout >}}

## Getting Started

Choose the package you need:

- Need container/data utilities: Start with [collectionx](/docs/collectionx)
- Need authentication/authorization: Start with [authx](/docs/authx)
- Need configuration management: Start with [configx](/docs/configx)
- Need modular app composition and typed lifecycle management: Start with [dix](/docs/dix)
- Need event bus: Start with [eventx](/docs/eventx)
- Need HTTP routing: Start with [httpx](/docs/httpx)
- Need logging: Start with [logx](/docs/logx)
- Need observability: Start with [observabilityx](/docs/observabilityx)

## Links

- [GitHub Repository](https://github.com/DaiYuANg/arcgo)
- [Go Module](https://pkg.go.dev/github.com/DaiYuANg/arcgo)
