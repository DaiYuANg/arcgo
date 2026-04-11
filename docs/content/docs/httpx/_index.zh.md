---
title: 'httpx'
linkTitle: 'httpx'
description: '多框架统一强类型 HTTP 路由'
weight: 5
---

## 概览

`httpx` 是构建在 Huma 之上的轻量 HTTP 服务组织层。
它提供一套稳定的 **server/group/endpoint** 组织与注册 API，让你在不同运行时（std/chi、gin、echo、fiber）之间保持一致的“强类型路由体验”，同时在需要时仍可直接访问 Huma 的高级能力。

## 安装

```bash
go get github.com/DaiYuANg/arcgo/httpx@latest
```

## 当前能力

- 跨适配器统一的强类型路由注册（`Get`/`Post`/`Put`/`Patch`/`Delete`...）
- 运行时适配器集成（`std`、`gin`、`echo`、`fiber`）
- endpoint/module 组织能力，支持作用域 registrar、默认元信息和受限 auto 语法糖（`Endpoint`、`Registrar`、`EndpointSpec`、`Auto`）
- 一等 OpenAPI 与文档控制（文档路由暴露由 adapter 负责）
- 强类型 SSE（`GetSSE`、`GroupGetSSE`）
- 基于 policy 的路由能力（`RouteWithPolicies`、`GroupRouteWithPolicies`）
- 条件请求（`If-Match`、`If-None-Match`、`If-Modified-Since`、`If-Unmodified-Since`）
- 面向路由模板的 observability helper（Prometheus / OpenTelemetry）
- `dix` 集成 helper（生命周期与监听 wiring）
- 直接访问 Huma 能力（`HumaAPI`、`OpenAPI`、`ConfigureOpenAPI`）
- 可选请求校验（`go-playground/validator`）
- 路由自省 API（用于测试与诊断）

## 包结构

- Core：`github.com/DaiYuANg/arcgo/httpx`
- Adapters：
  - `github.com/DaiYuANg/arcgo/httpx/adapter/std`
  - `github.com/DaiYuANg/arcgo/httpx/adapter/gin`
  - `github.com/DaiYuANg/arcgo/httpx/adapter/echo`
  - `github.com/DaiYuANg/arcgo/httpx/adapter/fiber`
- Optional：
  - `github.com/DaiYuANg/arcgo/httpx/middleware`
  - `github.com/DaiYuANg/arcgo/httpx/dix`
  - `github.com/DaiYuANg/arcgo/httpx/websocket`

## 文档导航（推荐阅读顺序）

- 最小可运行服务：[Getting Started](./getting-started)
- 适配器接入：[Adapters](./adapters)
- Endpoint 组织：[Endpoint Organization](./endpoint-organization)
- Middleware 与可观测性：[Middleware and Observability](./middleware-and-observability)
- `dix` 接入：[dix Integration](./dix-integration)
- OpenAPI 与文档：[OpenAPI and docs](./openapi-and-docs)

## 可运行示例（仓库）

- Quickstart：[examples/httpx/quickstart](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/quickstart)
- Adapters：
  - [examples/httpx/std](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/std)
  - [examples/httpx/gin](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/gin)
  - [examples/httpx/echo](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/echo)
  - [examples/httpx/fiber](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/fiber)
- Auth / organization：
  - [examples/httpx/auth](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/auth)
  - [examples/httpx/organization](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/organization)
- Streaming：
  - SSE：[examples/httpx/sse](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/sse)
  - Websocket：[examples/httpx/websocket](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/websocket)
- 条件请求：[examples/httpx/conditional](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/conditional)
- Endpoint 注册：[examples/httpx/endpoint](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/endpoint)
- `dix` 后端接线：[examples/dix/backend/http](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/backend/http)

## 定位（如何理解三层职责）

- `Huma`：typed operations、schema、OpenAPI/文档、middleware 模型
- `adapter/*`：运行时/路由器集成 + 框架原生 middleware 生态
- `httpx`：统一的服务组织 API + 暴露部分 Huma 能力
- `httpx/websocket`：轻量 websocket helper，直接在框架/router 层使用；它刻意不属于 typed route API
