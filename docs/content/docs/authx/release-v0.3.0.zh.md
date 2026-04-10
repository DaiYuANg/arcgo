---
title: 'authx v0.3.0（重构版）'
linkTitle: 'release v0.3.0'
description: 'authx 新版核心建模与 HTTP 集成说明'
weight: 4
---

## 概览

`authx v0.3.0` 是一次面向“统一抽象 + 多场景扩展”的重构版本。

核心变化：

- 认证与鉴权 API 明确拆分为 `Check` 与 `Can`
- `Engine` 不绑定具体认证方式（JWT/Session/OTP 等均可扩展）
- `ProviderManager` 支持多种 credential 类型并行注册
- 新增 `authx/http` 子包，覆盖 `std`/`gin`/`echo`/`fiber`
- 新增 `RequireFast` 与 `TypedGuard`，优化热路径与类型体验

## 新核心模型

新版核心是三层协作：

- `AuthenticationProvider[C]`: 处理某一类凭证 `C`
- `AuthenticationManager`: 负责选择并调用匹配 provider
- `Authorizer`: 专注授权决策（与认证解耦）

`Engine` 仅编排流程：

1. `Check(ctx, credential)` 返回 `AuthenticationResult`
2. `Can(ctx, AuthorizationModel)` 返回 `Decision`

## 典型调用方式

```go
engine := authx.NewEngine(
    authx.WithAuthenticationManager(
        authx.NewProviderManager(
            authx.NewAuthenticationProviderFunc(func(
                _ context.Context,
                in UsernamePassword,
            ) (authx.AuthenticationResult, error) {
                // verify...
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

## HTTP 集成（新增）

新增 `authx/http` 统一 Guard 层：

- `authx/http/std`
- `authx/http/gin`
- `authx/http/echo`
- `authx/http/fiber`

统一扩展点：

- `WithCredentialResolverFunc`
- `WithAuthorizationResolverFunc`

使用 std adapter（`chi + net/http`）时：

```go
guard := authhttp.NewGuard(
    engine,
    authhttp.WithCredentialResolverFunc(resolveCredential),
    authhttp.WithAuthorizationResolverFunc(resolveAuthorization),
)

router.Use(authstd.Require(guard))
// 或：router.Use(authstd.RequireFast(guard))
```

## 性能与可维护性更新

- 新增 core benchmark（含并行场景）
- 为 `std/gin/echo/fiber` 各自新增 middleware benchmark
- 热路径减少请求期对象构造，`RequireFast` 进一步降低分配
- `authx` 目录内 Go 文件按可维护性拆分，避免超长单文件

## 示例

- 通用示例：`authx/http/examples/shared`
- JWT 示例：`authx/http/examples/jwt`
- 框架示例：`authx/http/examples/std|gin|echo|fiber`

## 基准命令

```bash
go test ./authx -run ^$ -bench BenchmarkEngine -benchmem
go test ./authx/http/std -run ^$ -bench BenchmarkRequire -benchmem
go test ./authx/http/gin -run ^$ -bench BenchmarkRequire -benchmem
go test ./authx/http/echo -run ^$ -bench BenchmarkRequire -benchmem
go test ./authx/http/fiber -run ^$ -bench BenchmarkRequire -benchmem
```
