---
title: 'dix'
linkTitle: 'dix'
description: 'Strongly typed modular app framework built on do'
weight: 6
---

## dix

`dix` is a strongly typed, module-oriented application framework built on top of `do`.
It provides an immutable app spec, typed providers and invokes, lifecycle hooks, validation,
and a runtime model without forcing most users to deal with `do` directly.

## Current capabilities

- **Immutable spec**: `App` and `Module` are built as declarative specs.
- **Typed DI**: `ProviderN` registers typed constructors; `InvokeN` runs typed eager initialization.
- **Lifecycle**: `OnStart` / `OnStop` hooks with `Runtime.Start/Stop/StopWithReport`.
- **Validation**: `app.Validate()` fails on graph errors; `app.ValidateReport()` also exposes validation warnings for raw escape hatches.
- **Runtime**: container access, health checks, and diagnostics.
- **Advanced features**: named services, alias binding, transient providers, overrides, scopes via `dix/advanced`.

## Package layout

- Default path: `github.com/DaiYuANg/arcgo/dix`
- Advanced container features: `github.com/DaiYuANg/arcgo/dix/advanced`

## Documentation map

- Minimal module graph: [Getting Started](./getting-started)
- Health checks and HTTP handlers: [Health and lifecycle](./health-and-lifecycle)
- Fallible provider construction: [Error-returning providers](./error-providers)
- Release notes: [dix v0.3.0](./release-v0.3.0)
- Runnable example index: [dix examples](./examples)

## Install / Import

```bash
go get github.com/DaiYuANg/arcgo/dix@latest
```

## Key API surface (summary)

- `dix.New(name, ...)` / `dix.NewDefault(...)`
- `dix.NewModule(name, ...)`
- `dix.Modules(...)`, `dix.UseProfile(...)`, `dix.Version(...)`, `dix.UseLogger(...)`
- `dix.Providers(...)`, `dix.Hooks(...)`, `dix.Imports(...)`, `dix.Setups(...)`
- `dix.WithModules(...)`, `dix.WithProfile(...)`, `dix.WithVersion(...)`, `dix.WithLogger(...)`
- `dix.WithModuleProviders(...)`, `dix.WithModuleHooks(...)`, `dix.WithModuleImports(...)`
- `dix.WithModuleProvider(...)`, `dix.WithModuleHook(...)`, `dix.WithModuleImport(...)`
- `dix.Value(...)`, `dix.Invoke(...)`, `dix.ProviderN(...)`, `dix.OnStart(...)`, `dix.OnStop(...)`
- `advanced.Named(...)`, `advanced.Alias(...)`, `advanced.NamedAlias(...)`, `advanced.Transient(...)`, `advanced.Override(...)`
- `app.Validate()`, `app.ValidateReport()`, `app.Build()`, `app.Start(ctx)`, `app.RunContext(ctx)`
- `rt.Start(ctx)`, `rt.Stop(ctx)`, `rt.StopWithReport(ctx)`

## API style notes

- `dix` keeps the existing `WithModule*` option family for compatibility.
- `dix` also keeps the existing `WithProfile` / `WithVersion` / `WithLogger` / `WithModules` app option family for compatibility.
- New code can prefer the shorter module option aliases such as `Providers(...)`, `Hooks(...)`, `Imports(...)`, `Invokes(...)`, `Setups(...)`, `Description(...)`, and `Tags(...)`.
- For zero-dependency registrations, `Value(...)` and `Invoke(...)` reduce the remaining boilerplate on the core path.
- In `dix/advanced`, the shorter aliases such as `Named(...)`, `Alias(...)`, `Transient(...)`, and `Override(...)` keep the same semantics as the older explicit names.
- When you want the common build-then-start flow, prefer `app.Start(ctx)`; use `app.Build()` when you need an explicit pre-start runtime handle.
- When the caller owns cancellation or shutdown timing, prefer `app.RunContext(ctx)` over `app.Run()`.

## Validation model

- Use `app.Validate()` when you only care about hard failures.
- Use `app.ValidateReport()` when you also want to inspect warnings from raw providers, raw invokes, raw hooks, or raw setups.
- Typed `ProviderN` / `InvokeN` / `OnStart` / `OnStop` stay on the strict validation path.
- Raw escape hatches are still supported, but you should prefer the metadata-aware forms such as `RawProviderWithMetadata(...)`, `RawInvokeWithMetadata(...)`, `RawHookWithMetadata(...)`, `RawSetupWithMetadata(...)`, and `advanced.DoSetupWithMetadata(...)` so the validator can keep reasoning about dependencies and graph mutations.

## Integration guide

- **configx**: load typed config once, then provide it as dependencies in modules.
- **logx**: initialize one process logger and inject into service modules.
- **httpx**: do HTTP bootstrap in setup/hook stages; keep route registration in dedicated modules.
- **dbx / kvx**: isolate persistence setup into infra modules.

## Testing and benchmarks

```bash
go test ./dix/...
go test ./dix -run ^$ -bench . -benchmem
```

## Production notes

- Keep module boundaries domain-driven; avoid large all-in-one modules.
- Fail fast on validate/build errors before runtime start.
- Use `StopWithReport` when teardown visibility matters.
