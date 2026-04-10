---
title: 'authx v0.3.0 (Refactor)'
linkTitle: 'release v0.3.0'
description: 'Overview of the new authx core model and HTTP integrations'
weight: 4
---

## Overview

`authx v0.3.0` is a major refactor focused on unified abstractions and multi-scenario extensibility.

Key updates:

- Authentication and authorization are split into `Check` and `Can`
- `Engine` is no longer coupled to a specific auth mechanism
- `ProviderManager` can register providers for multiple credential types
- New `authx/http` package with `std`/`gin`/`echo`/`fiber` integrations
- New `RequireFast` and `TypedGuard` for hot-path and typed ergonomics

## New Core Model

The new core is composed of:

- `AuthenticationProvider[C]`: handles one credential type `C`
- `AuthenticationManager`: selects and executes matched provider
- `Authorizer`: performs authorization decisions independently

`Engine` orchestrates:

1. `Check(ctx, credential)` -> `AuthenticationResult`
2. `Can(ctx, AuthorizationModel)` -> `Decision`

## Typical Usage

```go
engine := authx.NewEngine(
    authx.WithAuthenticationManager(
        authx.NewProviderManager(
            authx.NewAuthenticationProviderFunc(func(
                _ context.Context,
                in UsernamePassword,
            ) (authx.AuthenticationResult, error) {
                return authx.AuthenticationResult{
                    Principal: authx.Principal{ID: in.Username},
                }, nil
            }),
        ),
    ),
    authx.WithAuthorizer(authx.AuthorizerFunc(func(
        _ context.Context,
        model authx.AuthorizationModel,
    ) (authx.Decision, error) {
        return authx.Decision{Allowed: true}, nil
    })),
)

result, err := engine.Check(ctx, UsernamePassword{Username: "alice", Password: "secret"})
decision, err := engine.Can(ctx, authx.AuthorizationModel{
    Principal: result.Principal,
    Action:    "query",
    Resource:  "order",
})
```

## New HTTP Integrations

`authx/http` now provides a unified Guard layer:

- `authx/http/std`
- `authx/http/gin`
- `authx/http/echo`
- `authx/http/fiber`

Unified extension points:

- `WithCredentialResolverFunc`
- `WithAuthorizationResolverFunc`

Std adapter usage (`chi + net/http`):

```go
guard := authhttp.NewGuard(
    engine,
    authhttp.WithCredentialResolverFunc(resolveCredential),
    authhttp.WithAuthorizationResolverFunc(resolveAuthorization),
)

router.Use(authstd.Require(guard))
// or: router.Use(authstd.RequireFast(guard))
```

## Performance and Maintainability

- Added core benchmarks (including parallel scenarios)
- Added middleware benchmarks for `std/gin/echo/fiber`
- Reduced request-path temporary object creation
- Added `RequireFast` path to further reduce allocations
- Split large files in `authx` for better maintainability

## Examples

- Shared example: `authx/http/examples/shared`
- JWT example: `authx/http/examples/jwt`
- Framework examples: `authx/http/examples/std|gin|echo|fiber`

## Benchmark Commands

```bash
go test ./authx -run ^$ -bench BenchmarkEngine -benchmem
go test ./authx/http/std -run ^$ -bench BenchmarkRequire -benchmem
go test ./authx/http/gin -run ^$ -bench BenchmarkRequire -benchmem
go test ./authx/http/echo -run ^$ -bench BenchmarkRequire -benchmem
go test ./authx/http/fiber -run ^$ -bench BenchmarkRequire -benchmem
```
