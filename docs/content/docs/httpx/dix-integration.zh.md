---
title: 'httpx dix Integration'
linkTitle: 'dix-integration'
description: '使用 httpx/dix 减少重复生命周期 wiring'
weight: 7
---

## `dix` 集成

`httpx/dix` 是给 `dix` 应用准备的一层小 helper。

它不是替代 `httpx` 本身，而是把这类重复 DI 接线收起来：

- 导入 HTTP 模块
- 提供 `httpx.ServerRuntime`
- 把启动 hook 绑定到 `Listen` 或 `ListenPort`
- 默认把停止 hook 绑定到 `Shutdown`

## 它提供什么

- `NewModule(...)`
- `WithListen(...)`
- `WithListenPort(...)`
- `WithListen1(...)`
- `WithListenPort1(...)`
- 默认的 `Shutdown()` stop hook

## 最小示例

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

这样常见的生命周期 wiring 就集中到一个地方，路由注册方式本身不变。

## 什么时候适合用

适合：

- 项目本来就在用 `dix`
- 大多数服务都遵循类似的 HTTP 生命周期模式
- 你想减少重复的 `OnStart` / `OnStop` hooks

不适合：

- 项目没有使用 `dix`
- 启动/停止逻辑高度定制，helper 反而不贴合

## 示例

- 可运行接线示例：[examples/dix/backend/http](https://github.com/DaiYuANg/arcgo/tree/main/examples/dix/backend/http)
