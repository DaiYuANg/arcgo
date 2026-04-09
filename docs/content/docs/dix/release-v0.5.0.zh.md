---
title: 'dix v0.5.0'
linkTitle: 'release v0.5.0'
description: '框架事件日志、logger 语法糖与更短的 setup API'
weight: 42
---

`dix v0.5.0` 这一版主要聚焦在框架日志和 setup 注册这两类公开 API 的易用性。

## 重点更新

- 增加了 `UseLogger0/1/Err0/Err1(...)`，让框架 logger 也能按 `dix` 一贯的 typed 风格从 DI 里解析。
- 增加了 `UseEventLogger(...)` 和 `UseEventLogger0/1/Err0/Err1(...)`，调用方可以完全接管 dix 内部的 build/start/stop/health/debug 日志输出。
- 当配置 `EventLogger` 后，dix 内部日志会优先走这条事件日志链，而不是再绕开它直接调用 `slog`。
- 增加了更短的 setup / hook 帮助入口，例如 `Setup0`、`SetupContainer`、`SetupLifecycle`、`Setup1..6`、`OnStartFunc`、`OnStopFunc`。
- 文档和示例已经同步切到 `UseLogger...` / `UseEventLogger...` 作为主推荐写法。

## 兼容性说明

- 现有的 `WithLogger(...)` 和 `WithLoggerFrom...` 仍然保留。
- `Observer` 仍然适合 metrics 等旁路订阅场景，但不再推荐作为主框架 logger 自定义入口。

## 验证

已通过：

```bash
go test ./dix/... ./examples/dix/basic ./examples/dix/metrics ./examples/dix/inspect ./examples/dix/override ./examples/dix/runtime_scope ./examples/dix/build_runtime ./examples/dix/build_failure ./examples/dix/advanced_do_bridge
```
