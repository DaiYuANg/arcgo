---
title: 'observabilityx v0.2.0'
linkTitle: 'release v0.2.0'
description: '声明式指标 spec 与更多 metric instrument 的 breaking 升级'
weight: 41
---

`observabilityx v0.2.0` 是一个 breaking release。这个版本把指标记录的核心模型从临时的 `AddCounter` / `RecordHistogram` 调用，改成了“先声明 spec，再通过 typed instrument 记录值”。

## 重点更新

- 核心 facade 改成声明式 metric spec：
  - `obs.Counter(spec).Add(...)`
  - `obs.UpDownCounter(spec).Add(...)`
  - `obs.Histogram(spec).Record(...)`
  - `obs.Gauge(spec).Set(...)`
- 新增一组一等 spec 类型：
  - `CounterSpec`
  - `UpDownCounterSpec`
  - `HistogramSpec`
  - `GaugeSpec`
- 新增通用 metric 声明 option：
  - `WithDescription(...)`
  - `WithUnit(...)`
  - `WithLabelKeys(...)`
- `Prometheus` 和 `OTel` 两个 backend 现在都通过同一套 facade 契约来约束 label schema。
- `configx`、`eventx`、`clientx`、`dix/metrics` 这些内部集成都已经迁到新的声明式模型。

## Breaking Changes

- 从 `observabilityx.Observability` 移除了直接写指标的方法：
  - `AddCounter(...)`
  - `RecordHistogram(...)`
- 现有自定义 `Observability` 实现需要补齐：
  - `Counter(...)`
  - `UpDownCounter(...)`
  - `Histogram(...)`
  - `Gauge(...)`
- 指标声明变成显式步骤，因此调用方需要预先定义 metric name、unit、description 和 label keys。

## 为什么这样改

- 声明式 spec 能让 Prometheus 的 label schema 更稳定、更容易校验。
- 更多 metric 类型能覆盖 inflight、queue depth 这类常见运行时信号。
- 现在的 API 更接近真实生产代码的写法：声明一次，重复记录。

## 验证

已通过：

```bash
go test ./observabilityx/... ./configx/... ./eventx/... ./clientx/... ./dix/... ./examples/observabilityx/...
```
