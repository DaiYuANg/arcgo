---
title: 'observabilityx'
linkTitle: 'observabilityx'
description: '可选可观测性抽象（OTel/Prometheus）'
weight: 7
---

## 概览

`observabilityx` 为 **日志 / 追踪 / 指标** 提供一层可选的统一门面。它的目标是让 arcgo 的各个包保持稳定 API，同时让可观测性后端保持可选、可组合。

从 `v0.2.0` 开始，指标 API 改成了“声明 typed spec + 获取 instrument + 再记录值”的模型，不再走临时的 `AddCounter` / `RecordHistogram` 写法。

## 安装

```bash
go get github.com/DaiYuANg/arcgo/observabilityx@latest
go get github.com/DaiYuANg/arcgo/observabilityx/otel@latest
go get github.com/DaiYuANg/arcgo/observabilityx/prometheus@latest
```

## 文档导航

- 版本说明：[observabilityx v0.2.0](./release-v0.2.0)
- 最小用法 + 多后端组合：[Getting Started](./getting-started)
- Prometheus 暴露 `/metrics`：[Prometheus metrics endpoint](./prometheus-metrics)
- OTel 后端说明：[OpenTelemetry backend](./otel-backend)

## 后端

- `observabilityx.Nop()` - 默认 no-op backend
- `observabilityx/otel` - OpenTelemetry backend（trace + metrics）
- `observabilityx/prometheus` - Prometheus backend（metrics + `/metrics` handler）

## 指标模型

- 先用 `NewCounterSpec`、`NewHistogramSpec`、`NewUpDownCounterSpec`、`NewGaugeSpec` 声明 spec。
- 再通过 `obs.Counter(...)`、`obs.Histogram(...)`、`obs.UpDownCounter(...)`、`obs.Gauge(...)` 拿到 instrument。
- 最后通过 instrument 记录值。

## 可运行示例（仓库）

- Multi backend: [examples/observabilityx/multi](https://github.com/DaiYuANg/arcgo/tree/main/examples/observabilityx/multi)

## 集成建议

- 与 `authx`、`eventx`、`configx`：把 backend 作为依赖注入进去，不要让业务包直接绑定某个遥测实现。
- 与 `httpx`：通过 Prometheus adapter 暴露稳定的 `/metrics` 端点。
- 与 `logx`：把日志和 span/trace 上下文、指标维度对齐起来。

## 生产建议

- 本地/开发环境先从 `Nop()` 开始，再按环境启用真实 backend。
- 控制 metric cardinality 和 label 维度规模。
- 优先使用声明式 metric spec，而不是动态拼 metric 名和 label 集。
- 优先显式组合 backend（`Multi`），不要依赖隐式全局状态。
