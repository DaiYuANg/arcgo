---
sidebar_position: 2
---

# Usage Guide

This page explains how to use `httpx` in detail.

## Core concepts

### Endpoint

An Endpoint maps one or more route handlers.

```go
type UserEndpoint struct {
    httpx.BaseEndpoint
}

func (e *UserEndpoint) ListUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    e.Success(w, map[string]interface{}{"users": []string{"Alice", "Bob"}})
    return nil
}
```

### BaseEndpoint helpers

- `Success(w, data)`
- `Error(w, code, message)`
- `NotFound(w, message)`
- `BadRequest(w, message)`
- `Param(w, r, name)`
- `Query(w, r, name)`
- `Header(w, r, name)`

### Adapter

Adapter bridges `httpx` and the concrete framework.

```go
ginAdapter := gin.New()
fiberAdapter := fiber.New()
echoAdapter := echo.New()
stdAdapter := std.New()
```

### Server

Server centralizes registration and startup.

```go
server := httpx.NewServer(
    httpx.WithAdapter(ginAdapter),
    httpx.WithBasePath("/api"),
    httpx.WithPrintRoutes(true),
)
```

## Define endpoints

```go
type UserEndpoint struct {
    httpx.BaseEndpoint
}

func (e *UserEndpoint) ListUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    e.Success(w, map[string]interface{}{
        "users": []string{"Alice", "Bob", "Charlie"},
    })
    return nil
}

func (e *UserEndpoint) GetUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    id := e.Param(w, r, "id")
    e.Success(w, map[string]interface{}{
        "id":   id,
        "name": "User " + id,
    })
    return nil
}
```

## Create server and register routes

```go
server := httpx.NewServer(
    httpx.WithAdapter(ginAdapter),
    httpx.WithBasePath("/api"),
    httpx.WithPrintRoutes(true),
)

_ = server.Register(&UserEndpoint{})
```

## Route rules

### Method mapping

| Method prefix | HTTP method |
|-----------|----------|
| `Get` | GET |
| `Post` | POST |
| `Put` | PUT |
| `Delete` | DELETE |
| `Patch` | PATCH |
| `Head` | HEAD |
| `Options` | OPTIONS |

### Custom routes

```go
func (e *UserEndpoint) Routes() []httpx.Route {
    return []httpx.Route{
        {Method: "GET", Path: "/users", Handler: e.ListUsers},
        {Method: "GET", Path: "/users/:id", Handler: e.GetUser},
        {Method: "POST", Path: "/users", Handler: e.CreateUser},
    }
}
```

## Request handling

```go
id := e.Param(w, r, "id")
page := e.QueryInt(w, r, "page", 1)
auth := e.Header(w, r, "Authorization")
```

```go
var input CreateUserInput
if err := e.ParseJSON(r, &input); err != nil {
    return e.BadRequest(w, "Invalid JSON")
}
```

## Responses

```go
e.Success(w, data)
e.SuccessWithCode(w, 201, data)

e.Error(w, 500, "Internal error")
e.BadRequest(w, "Invalid input")
e.NotFound(w, "User not found")
```

## Error logging

`httpx` logs internal failures with `slog` in these cases:

- JSON encode/write failures in `BaseEndpoint.JSON/Success/Error`
- OpenAPI JSON / docs HTML write failures in `huma.Service.RegisterHandler`

By default it uses `slog.Default()`. You can replace it globally:

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
slog.SetDefault(logger)
```

Or inject into Huma service directly:

```go
svc := huma.NewService(api, "My API", "1.0.0", "API docs").WithLogger(logger)
```
