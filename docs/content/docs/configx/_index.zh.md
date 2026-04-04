---
title: 'configx'
linkTitle: 'configx'
description: '分层配置加载与校验'
weight: 3
---

## configx

`configx` 是基于 `koanf` 与 `go-playground/validator` 构建的分层配置加载器。

它主要提供两种使用方式：

- **强类型加载**：`LoadT[T]` / `LoadTErr[T]` 反序列化到结构体，并可选执行校验。
- **动态配置**：`LoadConfig` 返回 `*configx.Config`，用于按路径读取（`GetString`、`Exists`、`All`、`Unmarshal`）。

配置源按优先级合并：后加载的源会覆盖先加载的源。默认顺序为 `dotenv → file → env → args`。

## 当前能力

- `.env` 加载（`WithDotenv`、`WithIgnoreDotenvError`）
- 文件加载（YAML/JSON/TOML）（`WithFiles`）
- 环境变量（`WithEnvPrefix`、`WithEnvSeparator`）
- 命令行参数与 flag（`WithArgs`、`WithOSArgs`、`WithFlagSet`、`WithCommandLineFlags`、`WithArgsNameFunc`）
- 显式合并顺序（`WithPriority`）
- 默认值（`WithDefaults`、`WithDefaultsTyped`、`WithTypedDefaults`）
- 可选校验（`WithValidateLevel`、`WithValidator`）
- 可选可观测性（`WithObservability`）
- 热更新（`Watcher`：`NewWatcher` / `Watch`）

## 文档导航

- 强类型最小示例（含校验）：[快速开始](./getting-started)
- 文件、环境变量、命令行参数与合并顺序：[配置源与优先级](./sources-and-priority)
- 自定义校验器与动态访问：[校验与动态访问](./validation-and-dynamic)

## Install / Import

```bash
go get github.com/DaiYuANg/arcgo/configx@latest
```

## 核心 API（摘要）

- `Load(out, opts...)`：加载到结构体指针
- `LoadT[T](opts...)` / `LoadTErr[T](opts...)`：强类型加载
- `LoadConfig(opts...)`：返回 `*Config` 以便按路径访问
- `New(opts...)` / `NewT[T](opts...)`：构造可复用 Loader
- `NewWatcher(opts...)` / `Watch(ctx, ...)`：热更新

## 集成指南

- **dix**：启动时加载一次，通过模块 provider 注入强类型配置。
- **httpx**：用配置驱动监听地址、TLS 与中间件开关。
- **dbx / kvx**：集中管理 DSN 与后端选项，并按环境覆盖。
- **logx / observabilityx**：外置日志级别与埋点开关。

## 示例（仓库）

- [configx/examples/observability](https://github.com/DaiYuANg/arcgo/tree/main/configx/examples/observability)
