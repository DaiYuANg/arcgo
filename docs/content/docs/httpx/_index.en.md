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
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
)

type HealthOutput struct {
    Body struct {
        Status string `json:"status"`
    }
}

func main() {
    router := chi.NewMux()
    router.Use(middleware.Logger, middleware.Recoverer)
    a := std.New(router)

    s := httpx.New(
        httpx.WithAdapter(a),
        httpx.WithBasePath("/api"),
        httpx.WithOpenAPIInfo("My API", "1.0.0", "Service API"),
    )

    _ = httpx.Get(s, "/health", func(ctx context.Context, input *struct{}) (*HealthOutput, error) {
        out := &HealthOutput{}
        out.Body.Status = "ok"
        return out, nil
    })

    _ = s.ListenPort(8080)
}
```

## Core API

### Server

- `New(...)`
- `WithAdapter(...)`
- `WithBasePath(...)`
- `WithValidation()` / `WithValidator(...)`
- `WithPanicRecover(...)`
- `WithAccessLog(...)`
- `Listen(addr)`
- `ListenPort(port)`
- `Shutdown()`
- `HumaAPI()`
- `OpenAPI()`
- `ConfigureOpenAPI(...)`
- `PatchOpenAPI(...)`
- `UseHumaMiddleware(...)`

### Documentation / OpenAPI

Documentation routes are configured on the adapter at construction time:

```go
a := std.New(nil, adapter.HumaOptions{
    DocsPath:     "/reference",
    OpenAPIPath:  "/spec",
    SchemasPath:  "/schemas",
    DocsRenderer: httpx.DocsRendererScalar,
})

s := httpx.New(
    httpx.WithAdapter(a),
    httpx.WithOpenAPIInfo("Arc API", "1.0.0", "Service API"),
)
```

OpenAPI patching:

```go
s.ConfigureOpenAPI(func(doc *huma.OpenAPI) {
    doc.Tags = append(doc.Tags, &huma.Tag{Name: "internal"})
})
```

Notes:

- `WithOpenAPIInfo(...)` still patches OpenAPI metadata.
- Documentation route exposure is adapter-owned and set when constructing the adapter.
- For `std`/`chi`, register `chi` middlewares on the router before passing it to `std.New(...)`.
- To disable docs routes, pass `adapter.HumaOptions{DisableDocsRoutes: true}`.
- Supported built-in renderers:
  - `httpx.DocsRendererStoplightElements`
  - `httpx.DocsRendererScalar`
  - `httpx.DocsRendererSwaggerUI`

### Security / Components / Global Parameters

```go
s := httpx.New(
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

- `std`: `router := chi.NewMux(); router.Use(...); adapter := std.New(router, ...)`
- `gin`: `adapter.Router().Use(...)`
- `echo`: `adapter.Router().Use(...)`
- `fiber`: `adapter.Router().Use(...)`

Typed handler operation control stays at the `httpx` layer:

- `WithPanicRecover(...)` controls panic recovery for typed `httpx` handlers
- `WithAccessLog(...)` controls request logging via server logger

Runtime listener setup (like read/write/idle timeouts and max header bytes) is an adapter concern and should be configured on the adapter or underlying server library, not via `httpx/options.ServerOptions`.

## Logging

`httpx` logging is intentionally divided between layers:

- `httpx.WithLogger(...)` configures route registration, access log, and typed-handler logging in `httpx`
- Framework-native loggers and middleware remain framework concerns
- Thin adapters do not expose a separate bridge-logger API

In practice this means:

- Use `httpx.WithLogger(...)` for `httpx`-level logs
- Continue configuring `chi` / `gin` / `echo` / `fiber` logging middleware on the adapter router or engine/app
- For `std`/`chi`, add `chi.Use(...)` middleware before constructing `std.New(...)`

## Adapter Build

Adapters are thin wrappers around the official Huma integrations.

They are responsible for:

- accepting or creating the native router/app
- applying `adapter.HumaOptions` for docs/OpenAPI route exposure
- letting `httpx.Server` provide convenience `Listen(...)`, `ListenPort(...)`, and `Shutdown()`

```go
stdAdapter := std.New(nil, adapter.HumaOptions{
    DocsPath:     "/reference",
    OpenAPIPath:  "/spec",
    DocsRenderer: httpx.DocsRendererSwaggerUI,
})

ginAdapter := gin.New(existingEngine, adapter.HumaOptions{
    DisableDocsRoutes: true,
})
```

If you need framework-specific server tuning, run the framework directly with the native `Router()` / `App()`. `httpx` no longer standardizes timeout knobs.

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
opts.HumaVersion = "1.0.0"
opts.HumaDescription = "Service API"
opts.EnablePanicRecover = true
opts.EnableAccessLog = true

a := std.New(nil, adapter.HumaOptions{
    DocsPath:     "/reference",
    DocsRenderer: httpx.DocsRendererSwaggerUI,
})

s := httpx.New(append(opts.Build(), httpx.WithAdapter(a))...)
```

## Test Mode

```go
a := std.New(nil)
s := httpx.New(httpx.WithAdapter(a))

req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
rec := httptest.NewRecorder()
a.Router().ServeHTTP(rec, req)

if rec.Code != http.StatusOK {
    t.Fatal(rec.Code)
}
```

## FAQ

### Do I have to use Huma-style input structs?

Yes, for typed route handlers in this package.

### Can I still access the raw Huma API?

Yes. Use `HumaAPI()`, `OpenAPI()`, or `Group(...).HumaGroup()`.

### Should `httpx` also wrap adapter middleware?

No. Keep adapter-native middleware on the adapter itself, and use `httpx` for Huma endpoint middleware and service organization.

## Examples

- Quickstart: `go run ./examples/httpx/quickstart`
  - Minimal typed routing + validation + base path
- Auth: `go run ./examples/httpx/auth`
  - Security schemes, global headers, and typed auth header binding
  - See [`examples/httpx/auth/README.md`](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/auth)
- Organization: `go run ./examples/httpx/organization`
  - Documentation paths, security, global headers, and group defaults
  - See [`examples/httpx/organization/README.md`](https://github.com/DaiYuANg/arcgo/tree/main/examples/httpx/organization)
- SSE: `go run ./examples/httpx/sse`
  - Typed event streaming over `text/event-stream`
- Conditional Requests: `go run ./examples/httpx/conditional`
  - ETag and Last-Modified based precondition checks
