---
title: '返回错误的 Provider'
linkTitle: 'error providers'
description: '当 provider 构造可能失败时，使用带 Err 后缀的 dix helper'
weight: 25
---

## 返回错误的 Provider

当服务构造过程本身可能失败，并且你希望这个失败沿着正常的 `dix` 解析链路向上传递时，使用带 `Err` 后缀的 API。

## 为什么单独命名

Go 不支持仅按返回值做重载。这意味着 `Provider0(func() T)` 不能同时再接受 `func() (T, error)`，否则 API 会变得含糊。

所以 `dix` 保留原有 API 不变，新增一组显式的 `Err` 变体。

## 核心 API

- `dix.ProviderErr0..6`
- `dixadvanced.NamedProviderErr0..3`
- `dixadvanced.TransientProviderErr0..1`
- `dixadvanced.NamedTransientProviderErr0..1`

## Scoped 与 Override API

- `dixadvanced.ProvideScopedErr0..3`
- `dixadvanced.ProvideScopedNamedErr0..3`
- `dixadvanced.OverrideErr0..1`
- `dixadvanced.NamedOverrideErr0..1`
- `dixadvanced.OverrideTransientErr0..1`
- `dixadvanced.NamedOverrideTransientErr0..1`

## 示例

```go
app := dix.NewApp("app",
    dix.NewModule("infra",
        dix.WithModuleProviders(
            dix.ProviderErr1(func(cfg Config) (*DB, error) {
                return OpenDB(cfg.DSN)
            }),
        ),
    ),
)
```

## Scoped 示例

```go
scope := advanced.Scope(rt, "request-42", func(injector do.Injector) {
    advanced.ProvideScopedNamedErr0(injector, "tenant.default", func() (string, error) {
        return resolveTenantFromRequest()
    })
})
```

## Override 示例

```go
module := dix.NewModule("test",
    dix.WithModuleSetups(
        advanced.OverrideErr0(func() (*Config, error) {
            return loadFixtureConfig()
        }),
    ),
)
```

## 使用准则

- 构造过程纯粹且不会失败时，用 `ProviderN`。
- 构造过程涉及 I/O、解析、校验等可能失败的步骤时，用 `ProviderErrN`。
- 在 `advanced` helper 里也保持 `Err` 后缀，让调用点一眼可见其失败语义。
