---
title: 'dix v0.3.0'
linkTitle: 'release v0.3.0'
description: '公开 API 简化、运行时启动语法糖与 advanced 快捷入口'
weight: 4
---

`dix v0.3.0` 是一个以公开 API 易用性为重点的功能版本。目标不是替换原有显式 API，而是在保持兼容的前提下，把最常见的 typed 应用流程进一步缩短。

## 重点更新

- 增加了 `app.Start(ctx)`，覆盖最常见的 build 后立即启动路径。
- 增加了 `app.RunContext(ctx)`，让调用方可以通过自己的 context 控制关闭，而不只依赖 `Run()` 的信号处理。
- 增加了更短的 `App` option 写法，例如 `Modules(...)`、`UseProfile(...)`、`Version(...)`、`UseLogger(...)`。
- 增加了更短的 `Module` option 写法，例如 `Providers(...)`、`Hooks(...)`、`Imports(...)`、`Invokes(...)`、`Setups(...)`。
- 增加了零依赖快捷注册：`dix.Value(...)` 与 `dix.Invoke(...)`。
- 为 `dix/advanced` 增加了快捷入口，例如 `Named(...)`、`Alias(...)`、`NamedAlias(...)`、`Transient(...)`、`Override(...)`。

## 内部优化

- `App` 现在会缓存展开后的模块图以及 validation/build plan，而不是在每次 `Build()` 和 `ValidateReport()` 时重复计算。
- provider、invoke、lifecycle 三条路径现在复用了共享的依赖解析 helper。
- `dix` 内部进一步向仓库现有的 `collectionx` 风格收敛，统一 list/map/reduce 的实现方式。

## 兼容性

- 现有的 `With*` App option 仍然保留。
- 现有的 `WithModule*` option 仍然保留。
- 现有的 `ProviderN`、`InvokeN`、`OnStart`、`OnStop` 以及 `advanced` 显式 API 仍然保留。
- 这一版是加法升级，旧代码理论上无需迁移即可继续编译。

## 推荐写法

- 需要“拿到已启动 runtime”时，优先使用 `app.Start(ctx)`。
- 调用方自己管理取消与关闭时机时，优先使用 `app.RunContext(ctx)`。
- 新代码可以优先使用短 option 写法；兼容性敏感或历史代码路径仍可继续使用 `With*` API。
- 当注册本身没有依赖时，优先使用 `Value(...)` / `Invoke(...)` 以及 `advanced` 的短入口，减少不必要的样板。

## 验证

已通过：

```bash
go test ./dix/...
go test ./examples/dix/transient ./examples/dix/named_alias ./examples/dix/aggregate_params
```
