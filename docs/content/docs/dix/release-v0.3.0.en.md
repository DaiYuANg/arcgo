---
title: 'dix v0.3.0'
linkTitle: 'release v0.3.0'
description: 'Public API simplification, runtime startup sugar, and advanced shortcut APIs'
weight: 4
---

`dix v0.3.0` is a feature release focused on making the typed application flow shorter without removing the explicit APIs that already existed.

## Highlights

- Added `app.Start(ctx)` for the common build-then-start path.
- Added `app.RunContext(ctx)` so callers can control shutdown through their own context instead of signal-only `Run()`.
- Added shorter `App` option aliases such as `Modules(...)`, `UseProfile(...)`, `Version(...)`, and `UseLogger(...)`.
- Added shorter `Module` option aliases such as `Providers(...)`, `Hooks(...)`, `Imports(...)`, `Invokes(...)`, and `Setups(...)`.
- Added zero-dependency shortcut registrations: `dix.Value(...)` and `dix.Invoke(...)`.
- Added `dix/advanced` shortcut APIs such as `Named(...)`, `Alias(...)`, `NamedAlias(...)`, `Transient(...)`, and `Override(...)`.

## Internal Improvements

- `App` now caches the flattened module graph and validation/build plan instead of recomputing it on every `Build()` and `ValidateReport()` call.
- Shared dependency resolution helpers are reused across provider, invoke, and lifecycle registration paths.
- `dix` internals were aligned further with the repository’s `collectionx` style for lists, maps, and reductions.

## Compatibility

- Existing `With*` app options remain supported.
- Existing `WithModule*` options remain supported.
- Existing `ProviderN`, `InvokeN`, `OnStart`, `OnStop`, and advanced explicit APIs remain supported.
- This release is additive: old code should continue to compile without migration.

## Recommended Style

- Prefer `app.Start(ctx)` when you want an immediately started runtime.
- Prefer `app.RunContext(ctx)` when the caller already owns cancellation.
- Prefer short option aliases in new code, while keeping `With*` APIs for compatibility-sensitive call sites.
- Use `Value(...)` / `Invoke(...)` and the advanced shortcuts when the registration has no dependencies and the longer names add noise.

## Validation

Verified with:

```bash
go test ./dix/...
go test ./examples/dix/transient ./examples/dix/named_alias ./examples/dix/aggregate_params
```
