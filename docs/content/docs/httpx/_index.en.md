---
title: 'httpx'
linkTitle: 'httpx'
description: 'Multi-Framework Unified Strongly Typed HTTP Routing'
weight: 5
---

## httpx

`httpx` is a lightweight HTTP service organization layer built on top of Huma.

## Roadmap

- Module roadmap: [httpx roadmap](./roadmap)
- Global roadmap: [ArcGo roadmap](../roadmap)

## What You Get

- Unified typed route registration across adapters (`Get`, `Post`, `Put`, `Patch`, `Delete`...)
- Adapter-based runtime integration (`std`, `gin`, `echo`, `fiber`)
- First-class OpenAPI and documentation control
- Typed Server-Sent Events (SSE) route registration (`GetSSE`, `GroupGetSSE`)
- Policy-based route capabilities (`RouteWithPolicies`, `GroupRouteWithPolicies`)
- Conditional request handling (`If-Match`, `If-None-Match`, `If-Modified-Since`, `If-Unmodified-Since`)
- Direct Huma escape hatches (`HumaAPI`, `OpenAPI`, `ConfigureOpenAPI`)
- Group-level Huma middleware and operation customization
- Optional request validation via `go-playground/validator`
- Route introspection API for testing and diagnostics

## Positioning

`httpx` is not a heavy web framework, nor does it intend to replace Huma.
It provides a stable server/group/endpoint API surface while retaining direct access to Huma's advanced features.

The division of responsibilities is as follows:

- `Huma`: Typed operations, schemas, OpenAPI, documentation, middleware model
- `adapter/*`: Runtime, router integration, native middleware ecosystem
- `httpx`: Unified service organization API and Huma capability exposure

## Minimal Setup

```go
package main

import (
    "context"

    "github.com/DaiYuANg/arcgo/httpx"
    "github.com/DaiYuANg/arcgo/httpx/adapter/std"
    "github.com/go-chi/chi/v5/middleware"
)

type HealthOutput struct {
    Body struct {
        Status string `json:"status"`
    }
}

func main() {
    a := std.New()
    a.Router().Use(middleware.Logger, middleware.Recoverer)

    s := httpx.NewServer(
        httpx.WithAdapter(a),
        httpx.WithBasePath("/api"),
        httpx.WithOpenAPIInfo("My API", "1.0.0", "Service API"),
    )

    _ = httpx.Get(s, "/health", func(ctx context.Context, input *struct{}) (*HealthOutput, error) {
        out := &HealthOutput{}
        out.Body.Status = "ok"
        return out, nil
    })

    _ = s.ListenAndServe(":8080")
}
```

## Core API

### Server

- `NewServer(...)`
- `WithAdapter(...)`
- `WithBasePath(...)`
- `WithValidation()` / `WithValidator(...)`
- `WithPanicRecover(...)`
- `WithAccessLog(...)`
- `HumaAPI()`
- `OpenAPI()`
- `ConfigureOpenAPI(...)`
- `PatchOpenAPI(...)`
- `UseHumaMiddleware(...)`

### Documentation / OpenAPI

Build-time documentation configuration:

```go
s := httpx.NewServer(
    httpx.WithDocs(httpx.DocsOptions{
        Enabled:     true,
        DocsPath:    "/reference",
        OpenAPIPath: "/spec",
        SchemasPath: "/schemas",
        Renderer:    httpx.DocsRendererScalar,
    }),
)
```

Runtime documentation configuration:

```go
s.ConfigureDocs(func(d *httpx.DocsOptions) {
    d.DocsPath = "/docs/internal"
    d.OpenAPIPath = "/openapi/internal"
})
```

OpenAPI patching:

```go
s.ConfigureOpenAPI(func(doc *huma.OpenAPI) {
    doc.Tags = append(doc.Tags, &huma.Tag{Name: "internal"})
})
```

Notes:

- `WithOpenAPIInfo(...)` and `WithOpenAPIDocs(...)` still work.
- `ConfigureDocs(...)` now also updates adapter-managed doc routes.
- Supported built-in renderers:
  - `httpx.DocsRendererStoplightElements`
  - `httpx.DocsRendererScalar`
  - `httpx.DocsRendererSwaggerUI`

### Security / Components / Global Parameters

```go
s := httpx.NewServer(
    httpx.WithSecurity(httpx.SecurityOptions{
        Schemes: map[string]*huma.SecurityScheme{
            "bearerAuth": {
                Type:   "http",
                Scheme: "bearer",
            },
        },
        Requirements: []map[string][]string{
            {"bearerAuth": {}},
        },
    }),
)

s.RegisterComponentParameter("Locale", &huma.Param{
    Name: "locale",
    In:   "query",
    Schema: &huma.Schema{Type: "string"},
})

s.RegisterGlobalHeader(&huma.Param{
    Name:   "X-Request-Id",
    In:     "header",
    Schema: &huma.Schema{Type: "string"},
})
```

Available API:

- `RegisterSecurityScheme(...)`
- `SetDefaultSecurity(...)`
- `RegisterComponentParameter(...)`
- `RegisterComponentHeader(...)`
- `RegisterGlobalParameter(...)`
- `RegisterGlobalHeader(...)`
- `AddTag(...)`

### Groups

Basic grouping:

```go
api := s.Group("/v1")
_ = httpx.GroupGet(api, "/users/{id}", getUser)
_ = httpx.GroupPost(api, "/users", createUser)
```

Group-level Huma capabilities:

```go
api := s.Group("/admin")
api.UseHumaMiddleware(authMiddleware)
api.DefaultTags("admin")
api.DefaultSecurity(map[string][]string{"bearerAuth": {}})
api.DefaultParameters(&huma.Param{
    Name:   "X-Tenant",
    In:     "header",
    Schema: &huma.Schema{Type: "string"},
})
api.DefaultSummaryPrefix("Admin")
api.DefaultDescription("Administrative APIs")
```

Available group API:

- `HumaGroup()`
- `UseHumaMiddleware(...)`
- `UseOperationModifier(...)`
- `UseSimpleOperationModifier(...)`
- `UseResponseTransformer(...)`
- `DefaultTags(...)`
- `DefaultSecurity(...)`
- `DefaultParameters(...)`
- `DefaultSummaryPrefix(...)`
- `DefaultDescription(...)`

### Policy Route Registration

```go
_ = httpx.RouteWithPolicies(server, httpx.MethodGet, "/resources/{id}", handler,
    httpx.PolicyOperation[GetInput, GetOutput](huma.OperationTags("resources")),
    httpx.PolicyConditionalRead[GetInput, GetOutput](stateGetter),
)
```

Available policy route API:

- `RouteWithPolicies(...)`
- `GroupRouteWithPolicies(...)`
- `MustRouteWithPolicies(...)`
- `MustGroupRouteWithPolicies(...)`

### SSE

```go
httpx.MustRouteSSEWithPolicies(server, httpx.MethodGet, "/events", map[string]any{
    "tick": TickEvent{},
    "done": DoneEvent{},
}, func(ctx context.Context, input *StreamInput, send httpx.SSESender) {
    _ = send.Data(TickEvent{Index: 1})
    _ = send(httpx.SSEMessage{ID: 2, Data: DoneEvent{Message: "ok"}})
}, httpx.SSEPolicyOperation[StreamInput](huma.OperationTags("stream")))
```

Available SSE API:

- `RouteSSEWithPolicies(...)`
- `GroupRouteSSEWithPolicies(...)`
- `MustRouteSSEWithPolicies(...)`
- `MustGroupRouteSSEWithPolicies(...)`
- `SSEPolicyOperation(...)`
- `GetSSE(...)`
- `GroupGetSSE(...)`
- `MustGetSSE(...)`
- `MustGroupGetSSE(...)`

### Conditional Requests

```go
type GetInput struct {
    httpx.ConditionalParams
}

_ = httpx.RouteWithPolicies(server, httpx.MethodGet, "/resources/{id}", func(ctx context.Context, input *GetInput) (*Output, error) {
    return out, nil
}, httpx.PolicyConditionalRead[GetInput, Output](func(ctx context.Context, input *GetInput) (string, time.Time, error) {
    return currentETag, modifiedAt, nil
}))
```

Available conditional helpers:

- `ConditionalParams`
- `PolicyConditionalRead(...)`
- `PolicyConditionalWrite(...)`
- `OperationConditionalRead()`
- `OperationConditionalWrite()`

### Adapter Bridge Hook

```go
httpx.UseAdapter[adapter.LoggerConfigurer](server, func(cfg adapter.LoggerConfigurer) {
    cfg.SetLogger(logger)
})
```

### Graceful Shutdown Hooks (humacli)

```go
cli := humacli.New(func(hooks humacli.Hooks, opts *Options) {
    httpx.BindGracefulShutdownHooks(hooks, server, ":8888")
})
```

## Typed Input Patterns

```go
type GetUserInput struct {
    ID int `path:"id"`
}

type ListUsersInput struct {
    Page int `query:"page"`
    Size int `query:"size"`
}

type SecureInput struct {
    RequestID string `header:"X-Request-Id"`
}

type CreateUserInput struct {
    Body struct {
        Name  string `json:"name" validate:"required,min=2,max=64"`
        Email string `json:"email" validate:"required,email"`
    }
}
```

## Middleware Model

`httpx` uses a two-layer middleware model:

- Adapter-native middleware: Registered directly on adapter router/engine/app
- Huma middleware: Registered via `Server.UseHumaMiddleware(...)` or `Group.UseHumaMiddleware(...)`

Adapter middleware should remain adapter-native:

- `std`: `adapter.Router().Use(...)`
- `gin`: `adapter.Router().Use(...)`
- `echo`: `adapter.Router().Use(...)`
- `fiber`: `adapter.Router().Use(...)`

Typed handler operation control stays at the `httpx` layer:

- `WithPanicRecover(...)` controls panic recovery for typed `httpx` handlers
- `WithAccessLog(...)` controls request logging via server logger

Runtime listener setup (like read/write/idle timeouts and max header bytes) is an adapter concern and should be configured on the adapter or underlying server library, not via `httpx/options.ServerOptions`.

## Logging

`httpx` logger behavior is intentionally divided between layers:

- `httpx.WithLogger(...)` configures the `httpx.Server` logger
- Adapter logger configuration controls bridging layer errors emitted by `adapter/std`, `adapter/gin`, `adapter/echo`, and `adapter/fiber`
- Framework-native loggers and logging middleware remain framework concerns

In practice this means:

- Use `httpx.WithLogger(...)` for `httpx` routing/access log/route registration output
- Explicitly configure adapter logger when you want adapter bridge errors to use the same logger
- Continue configuring `chi` / `gin` / `echo` / `fiber` logging middleware on the adapter router or engine

`httpx` currently doesn't commit to fully replacing framework-native loggers.

## Adapter Build

Listener and bridging layer configuration belongs to the adapter, not `httpx.ServerOptions`.

For `net/http`-based adapters (like `std`, `gin`, and `echo`), use build-time adapter options:

```go
stdAdapter := std.NewWithOptions(std.Options{
    Logger: slogLogger,
    Server: std.ServerOptions{
        ReadTimeout:     15 * time.Second,
        WriteTimeout:    15 * time.Second,
        IdleTimeout:     60 * time.Second,
        ShutdownTimeout: 5 * time.Second,
        MaxHeaderBytes:  1 << 20,
    },
})
```

For `fiber`, timeout settings belong to the app configuration when the adapter creates the app:

```go
fiberAdapter := fiber.NewWithOptions(nil, fiber.Options{
    Logger: slogLogger,
    App: fiber.AppOptions{
        ReadTimeout:     15 * time.Second,
        WriteTimeout:    15 * time.Second,
        IdleTimeout:     60 * time.Second,
        ShutdownTimeout: 5 * time.Second,
    },
})
```

If you pass already-created framework objects, that framework object's own configuration remains authoritative.

## Introspection API

- `GetRoutes()`
- `GetRoutesByMethod(method)`
- `GetRoutesByPath(prefix)`
- `HasRoute(method, path)`
- `RouteCount()`

## Options Builder

You can build server options via `httpx/options`:

```go
opts := options.DefaultServerOptions()
opts.BasePath = "/api"
opts.HumaTitle = "Arc API"
opts.DocsPath = "/reference"
opts.DocsRenderer = httpx.DocsRendererSwaggerUI
opts.EnablePanicRecover = true
opts.EnableAccessLog = true

s := httpx.NewServer(append(opts.Build(), httpx.WithAdapter(a))...)
```

Use adapter build options alone for listener timeout and adapter logger configuration.

## Test Mode

```go
req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
rec := httptest.NewRecorder()
s.ServeHTTP(rec, req)

if rec.Code != http.StatusOK {
    t.Fatal(rec.Code)
}
```

## FAQ

### Do I have to use Huma-style input structs?

Yes, for typed route handlers in this package.

### Can I still access the raw Huma API?

Yes. Use `HumaAPI()`, `OpenAPI()`, and `HumaGroup()`.

### Should `httpx` also wrap adapter middleware?

No. Keep adapter-native middleware on the adapter itself, and use `httpx` for Huma endpoint middleware and service organization.

## Examples

- Quickstart: `go run ./httpx/examples/quickstart`
  - Minimal typed routing + validation + base path
- Auth: `go run ./httpx/examples/auth`
  - Security schemes, global headers, and typed auth header binding
  - See [`httpx/examples/auth/README.md`](https://github.com/DaiYuANg/arcgo/tree/main/httpx/examples/auth)
- Organization: `go run ./httpx/examples/organization`
  - Documentation paths, security, global headers, and group defaults
  - See [`httpx/examples/organization/README.md`](https://github.com/DaiYuANg/arcgo/tree/main/httpx/examples/organization)
- SSE: `go run ./httpx/examples/sse`
  - Typed event streaming over `text/event-stream`
- Conditional Requests: `go run ./httpx/examples/conditional`
  - ETag and Last-Modified based precondition checks
