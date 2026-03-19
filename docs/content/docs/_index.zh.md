---
title: 'ArcGo 文档'
description: '模块化 Go 后端基础设施工具集'
date: '2026-03-08T00:00:00+08:00'
draft: false
---

# ArcGo

**ArcGo** 是一个模块化的 Go 后端基础设施工具集。它由独立的包组成，因此你可以只采用你需要的部分。

## 快速开始

```bash
go get github.com/DaiYuANg/arcgo/{package}
```

## 包概览

| 包 | 作用 | 描述 |
| --- | --- | --- |
| [authx](./authx) | 认证与授权 | 多场景可扩展的认证与鉴权抽象层 |
| [clientx](./clientx) | 协议客户端 | 协议导向客户端（`http/tcp/udp`）+ 共享工程约束 |
| [collectionx](./collectionx) | 数据结构 | 泛型集合与并发安全结构 |
| [configx](./configx) | 配置管理 | 分层配置加载与校验 |
| [dix](./dix) | 应用框架 | 基于 `do` 的强类型模块化应用框架 |
| [eventx](./eventx) | 事件总线 | 进程内强类型事件总线 |
| [httpx](./httpx) | HTTP 路由 | 多框架统一强类型 HTTP 路由 |
| [kvx](./kvx) | Redis / Valkey 访问 | 强类型 Redis / Valkey 对象访问与 repository 层 |
| [sqltmplx](./sqltmplx) | SQL 模板 | SQL 优先的条件模板渲染与可选 parser 校验 |
| [logx](./logx) | 日志记录 | 结构化日志与 slog 互通 |
| [observabilityx](./observabilityx) | 可观测性 | 可选可观测性抽象（OTel/Prometheus） |

## 路线图

- 统一路线图（含全部模块）: [ArcGo roadmap](./roadmap)
- 模块细化路线图：
- [authx roadmap](./authx/roadmap)
- [clientx roadmap](./clientx/roadmap)
- [collectionx roadmap](./collectionx/roadmap)
- [configx roadmap](./configx/roadmap)
- [eventx roadmap](./eventx/roadmap)
- [httpx roadmap](./httpx/roadmap)
- [logx roadmap](./logx/roadmap)
- [observabilityx roadmap](./observabilityx/roadmap)

## 如何选择

- 需要容器/数据辅助：从 `collectionx` 开始
- 需要可扩展的认证/鉴权抽象：从 `authx` 开始
- 需要协议导向客户端（`http/tcp/udp`）并共享工程约束：从 `clientx` 开始
- 需要从 `.env` + 文件 + 环境变量加载配置：从 `configx` 开始
- 需要模块化应用组装、typed DI、生命周期和启动校验：从 `dix` 开始
- 需要进程内带类型负载的 pub/sub：从 `eventx` 开始
- 需要跨框架的统一类型化 HTTP 处理器：从 `httpx` 开始
- 需要强类型 Redis / Valkey 仓储与访问辅助：从 `kvx` 开始
- 需要 SQL 优先的动态查询模板和可选 parser 校验：从 `sqltmplx` 开始
- 需要结构化日志和轮转：从 `logx` 开始
- 需要可选的遥测抽象（OTel/Prometheus）：从 `observabilityx` 开始

## 典型组合

- **API 服务基线**: `httpx + configx + logx`
- **模块化应用基线**: `dix + configx + logx`
- **单体应用内事件驱动**: `eventx + logx`
- **Redis / Valkey 支撑服务**: `kvx + httpx + configx`
- **数据密集型工具/内部库**: `collectionx + configx`

## 常用命令

```bash
# 格式化代码
go tool task fmt

# 代码检查
go tool task lint

# 运行测试
go tool task test

# 全面检查
go tool task check
```

## Git 提交前 Hook

仓库使用 `lefthook`（通过 `go tool` 管理）。

每个 clone 只需执行一次安装：

```bash
go tool task git:hooks:install
```

手动执行 hook：

```bash
go tool task git:hooks:run
```

`pre-commit` hook 会执行：

- `go tool task fmt`
- `go tool task lint`

## 说明

- 代码注释统一为英文
- 中文文档统一使用 `_index.md` 文件

## 链接

- [GitHub 仓库](https://github.com/DaiYuANg/arcgo)
- [Go 模块](https://pkg.go.dev/github.com/DaiYuANg/arcgo)

