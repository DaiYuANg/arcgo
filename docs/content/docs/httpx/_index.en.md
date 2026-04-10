---
title: 'httpx'
linkTitle: 'httpx'
description: 'Multi-Framework Unified Strongly Typed HTTP Routing'
weight: 5
---

## Overview

`httpx` is a lightweight HTTP service organization layer built on top of Huma.
It gives you a stable **server/group/endpoint** API surface across multiple runtimes (std/chi, gin, echo, fiber), while still allowing direct access to Huma when you need it.

## Install

```bash
go get github.com/DaiYuANg/arcgo/httpx@latest
```

## Current capabilities

- Unified typed route registration across adapters (`Get`, `Post`, `Put`, `Patch`, `Delete`...)
- Adapter-based runtime integration (`std`, `gin`, `echo`, `fiber`)
- Endpoint/module organization with scoped metadata defaults (`Endpoint`, `GroupEndpoint`, `EndpointSpec`)
- First-class OpenAPI and documentation control (docs route exposure is adapter-owned)
- Typed Server-Sent Events (SSE) (`GetSSE`, `GroupGetSSE`)
- Policy-based route capabilities (`RouteWithPolicies`, `GroupRouteWithPolicies`)
- Conditional request handling (`If-Match`, `If-None-Match`, `If-Modified-Since`, `If-Unmodified-Since`)
- Route-aware observability helpers for Prometheus and OpenTelemetry
- `dix` integration helpers for lifecycle and listen wiring
- Direct Huma escape hatches (`HumaAPI`, `OpenAPI`, `ConfigureOpenAPI`)
- Optional request validation via `go-playground/validator`
- Route introspection API for testing and diagnostics

## Package layout

- Core: `github.com/DaiYuANg/arcgo/httpx`
- Adapters:
  - `github.com/DaiYuANg/arcgo/httpx/adapter/std`
  - `github.com/DaiYuANg/arcgo/httpx/adapter/gin`
  - `github.com/DaiYuANg/arcgo/httpx/adapter/echo`
  - `github.com/DaiYuANg/arcgo/httpx/adapter/fiber`
- Optional:
  - `github.com/DaiYuANg/arcgo/httpx/middleware`
  - `github.com/DaiYuANg/arcgo/httpx/dix`
  - `github.com/DaiYuANg/arcgo/httpx/websocket`

## Documentation map (recommended reading)

- Minimal typed server: [Getting Started](./getting-started)
- Adapter wiring: [Adapters](./adapters)
- Endpoint organization: [Endpoint Organization](./endpoint-organization)
- Middleware and observability: [Middleware and Observability](./middleware-and-observability)
- DI wiring: [dix Integration](./dix-integration)
- OpenAPI and docs: [OpenAPI and docs](./openapi-and-docs)

## Runnable examples (repository)

- Quickstart: [examples/httpx/quickstart](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/quickstart)
- Adapters:
  - [examples/httpx/std](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/std)
  - [examples/httpx/gin](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/gin)
  - [examples/httpx/echo](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/echo)
  - [examples/httpx/fiber](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/fiber)
- Auth / organization:
  - [examples/httpx/auth](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/auth)
  - [examples/httpx/organization](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/organization)
- Streaming:
  - SSE: [examples/httpx/sse](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/sse)
  - Websocket: [examples/httpx/websocket](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/websocket)
- Conditional requests: [examples/httpx/conditional](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/conditional)
- Endpoint registration: [examples/httpx/endpoint](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/endpoint)
- `dix` backend wiring: [examples/dix/backend/http](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/backend/http)

## Positioning (how to think about it)

- `Huma`: typed operations, schemas, OpenAPI/docs, middleware model
- `adapter/*`: runtime/router integration + native middleware ecosystem
- `httpx`: unified service organization API + exposes selected Huma capabilities
- `httpx/websocket`: lightweight websocket helper used directly at framework/router layer; it is intentionally not part of the typed route API
