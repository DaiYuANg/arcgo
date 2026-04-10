---
title: 'authx'
linkTitle: 'authx'
description: '多场景可扩展的认证与鉴权抽象层'
weight: 1
---

## authx

`authx` 是面向 HTTP、gRPC、CLI 等场景的 Go **认证**与**鉴权**抽象层，核心职责分离为：

- **认证** — `Engine.Check(ctx, credential)` 解析身份。
- **鉴权** — `Engine.Can(ctx, AuthorizationModel)` 基于主体与 action/resource 做策略判断。

`authx` **不绑定具体机制**：不内置密码哈希、JWT 解析、OTP 校验等。由业务定义凭证结构并实现 `AuthenticationProvider`；再接入自定义 `Authorizer`（RBAC/ABAC/Rego 等）。

## 当前能力

- **`Engine`** — 编排 `Check` / `Can`，可选 `Hook`。
- **`ProviderManager`** — 按凭证动态类型分发到泛型 `AuthenticationProvider[C]`。
- **`authx/http`** — `Guard` 从 `RequestInfo` 解析凭证与鉴权模型，再调用引擎。
- **HTTP 中间件** — `authx/http/std`（chi + net/http）、`gin`、`echo`、`fiber` 适配常见栈。
- **Context 辅助** — `WithPrincipal`、`PrincipalFromContext`、泛型 `PrincipalFromContextAs`。

## 包结构

- 核心 API：`github.com/DaiYuANg/arcgo/authx`
- HTTP Guard 与 `RequestInfo`：`github.com/DaiYuANg/arcgo/authx/http`
- std 中间件（`chi + net/http`）：`github.com/DaiYuANg/arcgo/authx/http/std`
- Gin：`github.com/DaiYuANg/arcgo/authx/http/gin`
- Echo：`github.com/DaiYuANg/arcgo/authx/http/echo`
- Fiber：`github.com/DaiYuANg/arcgo/authx/http/fiber`

## 文档导航

- 核心最小示例（`Check` / `Can`）：[快速开始](./getting-started)
- `Guard` + std adapter（`chi + net/http`）：[HTTP 集成](./http-integration)
- 版本说明（v0.3.0 重构）：[authx v0.3.0](./release-v0.3.0)

## 安装 / 导入

```bash
go get github.com/DaiYuANg/arcgo/authx@latest
go get github.com/DaiYuANg/arcgo/authx/http/std@latest
go get github.com/DaiYuANg/arcgo/authx/http/gin@latest
go get github.com/DaiYuANg/arcgo/authx/http/echo@latest
go get github.com/DaiYuANg/arcgo/authx/http/fiber@latest
```

## 核心 API（摘要）

| 组件 | 作用 |
| --- | --- |
| `Engine` | 执行 `Check` / `Can`，可挂 `Hook` |
| `ProviderManager` | 管理多个泛型 `AuthenticationProvider` |
| `AuthenticationProvider[C]` | `Authenticate(ctx, C)` → `AuthenticationResult` |
| `Authorizer` | `Authorize(ctx, AuthorizationModel)` → `Decision` |
| `AuthenticationResult` | 携带 `Principal`（`any`）与可选 `Details` |
| `AuthorizationModel` | `Principal`、`Action`、`Resource`、可选 `Context` |
| `Decision` | `Allowed`、`Reason`、`PolicyID` |

带完整 import 的可运行示例见 [快速开始](./getting-started)。

## HTTP 层（摘要）

`authhttp.NewGuard` 组合：

- **`WithCredentialResolverFunc`** — `(ctx, RequestInfo) → (credential any, err)`
- **`WithAuthorizationResolverFunc`** — `(ctx, RequestInfo, principal) → (AuthorizationModel, err)`

`Guard.Require` 依次执行 **Check** 与 **Can**。`authx/http/std` 就是 std adapter（`chi + net/http`），成功时会将 `Principal` 写入 `context`。

完整 std adapter（`chi + net/http`）示例见 [HTTP 集成](./http-integration)。

## 错误与行为模型

- `Check` 返回认证结果与错误；凭证无效应通过显式错误表达，而非“静默失败”。
- `Can` 返回决策与错误；策略引擎异常不应在不明示的情况下被当成默认拒绝。
- HTTP 中间件通过 `authx/http` 将失败映射为稳定状态码（如 `401` / `403`）；详见包内 `StatusCodeFromError` 等说明。

## 集成指南

- **httpx**：在路由组上挂载 guard；策略评估尽量放在服务层。
- **dix**：在模块中提供 `Engine`、provider、`Authorizer` 并注入到 HTTP 组装。
- **configx**：密钥、provider 开关、策略源外置。
- **logx / observabilityx**：记录 check/can 耗时与失败分类，避免记录明文密钥。

## 可运行示例（仓库）

- [examples/authx/jwt](https://github.com/DaiYuANg/arcgo/tree/main/examples/authx/jwt)
- [examples/authx/std](https://github.com/DaiYuANg/arcgo/tree/main/examples/authx/std)（Chi + shared）
- [examples/authx/gin](https://github.com/DaiYuANg/arcgo/tree/main/examples/authx/gin)
- [examples/authx/echo](https://github.com/DaiYuANg/arcgo/tree/main/examples/authx/echo)
- [examples/authx/fiber](https://github.com/DaiYuANg/arcgo/tree/main/examples/authx/fiber)
- 共享辅助：[examples/authx/shared](https://github.com/DaiYuANg/arcgo/tree/main/examples/authx/shared)

## 测试与基准

```bash
go test ./authx/...

# core
go test ./authx -run ^$ -bench BenchmarkEngine -benchmem

# middleware
go test ./authx/http/std -run ^$ -bench BenchmarkRequire -benchmem
go test ./authx/http/gin -run ^$ -bench BenchmarkRequire -benchmem
go test ./authx/http/echo -run ^$ -bench BenchmarkRequire -benchmem
go test ./authx/http/fiber -run ^$ -bench BenchmarkRequire -benchmem
```

## 生产注意事项

- 保持各接入层（HTTP/gRPC/CLI）下认证逻辑一致，避免语义漂移。
- 不在代码中硬编码密钥，统一从配置加载。
- 将策略加载视为启动关键路径，状态非法时应快速失败。
