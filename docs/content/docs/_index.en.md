---
title: 'ArcGo Documentation'
description: 'Modular Go Backend Infrastructure Toolkit'
date: '2026-03-08T00:00:00+08:00'
draft: false
---

# ArcGo

**ArcGo** is a modular Go backend infrastructure toolkit. It consists of independent packages, so you can adopt only what you need.

## Quick Start

```bash
go get github.com/DaiYuANg/archgo/{package}
```

## Package Overview

| Package | Purpose | Description |
| --- | --- | --- |
| [authx](./authx) | Authentication & Authorization | Extensible abstraction for multi-scenario authentication and authorization |
| [clientx](./clientx) | Protocol Clients | Protocol-oriented clients (`http/tcp/udp`) with shared engineering conventions |
| [collectionx](./collectionx) | Data Structures | Generic collections and concurrency-safe structures |
| [configx](./configx) | Configuration Management | Hierarchical configuration loading and validation |
| [eventx](./eventx) | Event Bus | In-process strongly typed event bus |
| [httpx](./httpx) | HTTP Routing | Multi-framework unified strongly typed HTTP routing |
| [logx](./logx) | Logging | Structured logging with slog interoperability |
| [observabilityx](./observabilityx) | Observability | Optional observability abstraction (OTel/Prometheus) |

## Roadmap

- Unified roadmap (all modules): [ArcGo roadmap](./roadmap)
- Module-level roadmap details:
- [authx roadmap](./authx/roadmap)
- [clientx roadmap](./clientx/roadmap)
- [collectionx roadmap](./collectionx/roadmap)
- [configx roadmap](./configx/roadmap)
- [eventx roadmap](./eventx/roadmap)
- [httpx roadmap](./httpx/roadmap)
- [logx roadmap](./logx/roadmap)
- [observabilityx roadmap](./observabilityx/roadmap)

## How to Choose

- Need container/data utilities: Start with `collectionx`
- Need an extensible authentication/authorization abstraction: Start with `authx`
- Need protocol-oriented clients (`http/tcp/udp`) with shared conventions: Start with `clientx`
- Need configuration loading from `.env` + files + environment variables: Start with `configx`
- Need in-process typed pub/sub: Start with `eventx`
- Need unified typed HTTP handlers across frameworks: Start with `httpx`
- Need structured logging with rotation: Start with `logx`
- Need optional telemetry abstraction (OTel/Prometheus): Start with `observabilityx`

## Typical Combinations

- **API Service Baseline**: `httpx + configx + logx`
- **Event-driven within monolith**: `eventx + logx`
- **Data-intensive tools/internal libraries**: `collectionx + configx`

## Common Commands

```bash
# Format code
go tool task fmt

# Lint code
go tool task lint

# Run tests
go tool task test

# Full check
go tool task check
```

## Pre-commit Git Hook

The repository uses `lefthook` (managed via `go tool`).

Install once per clone:

```bash
go tool task git:hooks:install
```

Run hooks manually:

```bash
go tool task git:hooks:run
```

The `pre-commit` hook runs:

- `go tool task fmt`
- `go tool task lint`

## Notes

- Code comments are unified in English
- Chinese documentation uses `_index.md` files

## Links

- [GitHub Repository](https://github.com/DaiYuANg/archgo)
- [Go Module](https://pkg.go.dev/github.com/DaiYuANg/archgo)
