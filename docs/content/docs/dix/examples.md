---
title: 'dix examples'
linkTitle: 'examples'
description: 'Runnable examples for dix'
weight: 10
---

## dix Examples

This page collects the runnable `examples/dix` programs and maps them to the API surface they demonstrate.

## Run Locally

Run from the `examples/dix` module:

```bash
cd examples/dix
go run ./basic
go run ./runtime_scope
go run ./inspect
```

## Core Examples

| Example | Focus | Directory |
| --- | --- | --- |
| `basic` | immutable app spec, `app.Start(ctx)`, health checks, `logx` integration | [examples/dix/basic](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/basic) |
| `aggregate_params` | provider graph composition with multiple typed dependencies | [examples/dix/aggregate_params](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/aggregate_params) |
| `build_runtime` | explicit `Build()` to `Runtime` flow | [examples/dix/build_runtime](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/build_runtime) |
| `build_failure` | validation/build failure behavior | [examples/dix/build_failure](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/build_failure) |

## Advanced Examples

| Example | Focus | Directory |
| --- | --- | --- |
| `advanced_do_bridge` | explicit `do` bridge setup | [examples/dix/advanced_do_bridge](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/advanced_do_bridge) |
| `named_alias` | named services and typed alias binding | [examples/dix/named_alias](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/named_alias) |
| `runtime_scope` | request-like runtime scope and scoped providers | [examples/dix/runtime_scope](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/runtime_scope) |
| `transient` | transient provider semantics | [examples/dix/transient](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/transient) |
| `override` | structured overrides | [examples/dix/override](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/override) |
| `inspect` | runtime inspection and diagnostics | [examples/dix/inspect](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/inspect) |

## Full-stack backend

End-to-end service shape: **configx** (dotenv → file → env), **logx**, **eventx**, **httpx** (chi + Huma), **dix**, **dbx** (SQLite). Layout: `config`, `domain`, `repo`, `service`, `api`, `event`, `db`.

| Example | Focus | Directory |
| --- | --- | --- |
| `backend` | integrated HTTP API + DI + SQLite persistence | [examples/dix/backend](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/backend) |

```bash
cd examples/dix
go run ./backend
# Optional: APP_SERVER_PORT=3000 APP_DB_DSN=file:app.db
# Open http://localhost:8080/docs
```

From the repository root:

```bash
go run ./examples/dix/backend
```

## Example: Basic App Composition

```go
app := dix.New(
    "basic",
    dix.UseLogger(logger),
    dix.WithModule(
        dix.NewModule("config",
            dix.Providers(dix.Provider0(func() Config { return Config{Port: 8080} })),
        ),
    ),
)

if err := app.Validate(); err != nil {
    panic(err)
}

rt, err := app.Start(context.Background())
if err != nil {
    panic(err)
}
defer func() {
    _, _ = rt.StopWithReport(context.Background())
}()
```

## Example: Validation Report For Raw Bridges

```go
report := app.ValidateReport()
if err := report.Err(); err != nil {
    panic(err)
}
for _, warning := range report.Warnings {
    logger.Warn("validation warning", "kind", warning.Kind, "module", warning.Module, "label", warning.Label)
}
```

Use this path when the module graph intentionally includes raw bridges.

## Example: Declared Raw Metadata

```go
module := dix.NewModule("bridge",
    dix.Providers(
        dix.Provider0(func() Config { return Config{Port: 8080} }),
        dix.RawProviderWithMetadata(func(c *dix.Container) {
            dix.ProvideValueT(c, &Server{})
        }, dix.ProviderMetadata{
            Label:        "RawServerProvider",
            Output:       dix.TypedService[*Server](),
            Dependencies: []dix.ServiceRef{dix.TypedService[Config]()},
        }),
    ),
    dix.Setups(
        advanced.DoSetupWithMetadata(func(raw do.Injector) error {
            _ = raw
            return nil
        }, dix.SetupMetadata{
            Label:         "RawBridgeSetup",
            Dependencies:  []dix.ServiceRef{dix.TypedService[Config]()},
            GraphMutation: true,
        }),
    ),
)
```

Declaring metadata keeps raw integration possible without making validation completely blind.

## Example: Runtime Scope

```go
requestScope := advanced.Scope(rt, "request-42", func(injector do.Injector) {
    advanced.ProvideScopedValue(injector, RequestContext{RequestID: "req-42"})
    advanced.ProvideScoped2(injector, func(cfg AppConfig, req RequestContext) ScopedService {
        return ScopedService{Config: cfg, Request: req}
    })
})

svc, err := advanced.ResolveScopedAs[ScopedService](requestScope)
if err != nil {
    panic(err)
}
fmt.Println(svc.Request.RequestID)
```

## Example: Error-returning Providers

```go
app := dix.NewApp("errors",
    dix.NewModule("errors",
        dix.Providers(
            dix.ProviderErr0(func() (*Config, error) {
                return loadConfig()
            }),
        ),
        dix.Setups(
            advanced.OverrideErr0(func() (*Config, error) {
                return loadConfigFromFixture()
            }),
        ),
    ),
)

requestScope := advanced.Scope(rt, "request-42", func(injector do.Injector) {
    advanced.ProvideScopedNamedErr0(injector, "tenant.default", func() (string, error) {
        return resolveTenantFromRequest()
    })
})
```

Use the `Err` suffixed helpers when construction can fail and the failure should flow through normal resolution.

## Example: Fine-Grained Inspection

```go
provided := advanced.ListProvidedServices(rt)
deps := advanced.ExplainNamedDependencies(rt, "tenant.default")

fmt.Println("provided services:", len(provided))
fmt.Println("tenant graph known:", deps["tenant.default"] != "")
```

Use the fine-grained inspection helpers when you only need one diagnostic view.
`InspectRuntime(...)` remains convenient, but it is the heavier aggregation path.

For caller-controlled shutdown, the app-level shortcut is:

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

if err := app.RunContext(ctx); err != nil {
    panic(err)
}
```
