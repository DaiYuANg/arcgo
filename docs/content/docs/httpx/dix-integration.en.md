---
title: 'httpx dix Integration'
linkTitle: 'dix-integration'
description: 'Use httpx/dix to reduce repeated lifecycle wiring'
weight: 7
---

## `dix` integration

`httpx/dix` is a small helper package for `dix`-based applications.

Its goal is not to replace `httpx` itself. It reduces repeated DI wiring around:

- importing the HTTP module
- providing `httpx.ServerRuntime`
- binding start hooks to `Listen` or `ListenPort`
- binding a default stop hook to `Shutdown`

## What it gives you

- `NewModule(...)`
- `WithListen(...)`
- `WithListenPort(...)`
- `WithListen1(...)`
- `WithListenPort1(...)`
- default `Shutdown()` stop hook

## Minimal example

```go
httpModule := httpxdix.NewModule(
	"http",
	di.Invoke(func(server httpx.ServerRuntime, svc UserService) {
		api.RegisterRoutes(server, svc)
	}),
	httpxdix.WithImports(baseModule),
	httpxdix.WithListenPort1(func(cfg Config) int {
		return cfg.HTTP.Port
	}),
)
```

This keeps the common lifecycle wiring in one place while leaving route registration unchanged.

## When to use it

Use `httpx/dix` if:

- the project already uses `dix`
- most services follow the same HTTP server lifecycle pattern
- you want fewer repeated `OnStart` / `OnStop` hooks

Do not use it if:

- the project does not use `dix`
- startup or shutdown logic is highly custom and the helper does not fit

## Example

- Runnable wiring example: [examples/dix/backend/http](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/backend/http)
