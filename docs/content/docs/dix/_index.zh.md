---
title: 'dix'
linkTitle: 'dix'
description: '基于 do 的强类型模块化应用框架'
weight: 6
---

## dix

`dix` 是构建在 `do` 之上的强类型、模块化应用框架。它提供不可变应用规格、typed provider/invoke、生命周期 hook、构建期校验，以及独立的运行时模型，同时默认路径不会强迫业务代码直接接触 `do`。

## 当前能力

- **不可变规格**：`App` 与 `Module` 以声明式 spec 组装。
- **强类型 DI**：`ProviderN` 注册强类型构造器；`InvokeN` 执行强类型预热初始化。
- **生命周期**：`OnStart` / `OnStop` hook，配合 `Runtime.Start/Stop/StopWithReport`。
- **校验**：`app.Validate()` 对依赖图错误快速失败；`app.ValidateReport()` 还会暴露 raw escape hatch 的校验警告。
- **运行时**：容器访问、健康检查、诊断入口。
- **高级能力**：named service、alias、transient、override、scope（见 `dix/advanced`）。

## 包结构

- 默认路径：`github.com/DaiYuANg/arcgo/dix`
- 高级容器能力：`github.com/DaiYuANg/arcgo/dix/advanced`

## 文档导航

- 最小模块图：[快速开始](./getting-started)
- 健康检查与 HTTP handler：[健康检查与生命周期](./health-and-lifecycle)
- 可失败的 provider 构造：[返回错误的 Provider](./error-providers)
- 可运行示例导航：[dix 示例](./examples)

## Install / Import

```bash
go get github.com/DaiYuANg/arcgo/dix@latest
```

## 核心 API（摘要）

- `dix.New(name, ...)` / `dix.NewDefault(...)`
- `dix.NewModule(name, ...)`
- `dix.WithModuleProviders(...)`、`dix.ProviderN(...)`
- `dix.WithModuleHooks(...)`、`dix.OnStart(...)`、`dix.OnStop(...)`
- `dix.WithModuleSetup(...)` / `dix.WithModuleSetups(...)`
- `app.Validate()`、`app.ValidateReport()`、`app.Build()`
- `rt.Start(ctx)`、`rt.Stop(ctx)`、`rt.StopWithReport(ctx)`

## 校验模型

- 只关心硬错误时，使用 `app.Validate()`。
- 还想查看 raw provider / invoke / hook / setup 带来的校验盲区时，使用 `app.ValidateReport()`。
- `ProviderN` / `InvokeN` / `OnStart` / `OnStop` 这些 typed API 仍然走严格校验路径。
- raw escape hatch 仍然可用，但更推荐 `RawProviderWithMetadata(...)`、`RawInvokeWithMetadata(...)`、`RawHookWithMetadata(...)`、`RawSetupWithMetadata(...)`、`advanced.DoSetupWithMetadata(...)` 这类带 metadata 的形式，让校验器继续理解依赖和图变更边界。

## 集成指南

- **configx**：启动加载一次强类型配置，并作为模块依赖注入。
- **logx**：初始化进程级 logger，并注入各服务模块。
- **httpx**：在 setup/hook 阶段完成 HTTP bootstrap；路由注册保持在独立模块内。
- **dbx / kvx**：把持久化初始化隔离到 infra 模块。

## 测试与基准

```bash
go test ./dix/...
go test ./dix -run ^$ -bench . -benchmem
```

## 生产注意

- 按领域边界拆分模块，避免把所有东西塞进一个大模块。
- 在 runtime start 前对 validate/build 错误快速失败。
- 需要更可观测的关闭诊断时优先使用 `StopWithReport`。
