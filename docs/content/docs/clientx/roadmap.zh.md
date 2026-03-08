---
title: 'roadmap'
linkTitle: 'roadmap'
description: 'clientx 路线图'
weight: 90
---

## clientx Roadmap（2026-03）

## 定位

`clientx` 是协议导向客户端层，不是重型 RPC 框架。

- 保持协议专属 API 显式（`http` 请求响应、`tcp` 流、`udp` 报文）。
- 统一工程约束而非调用形态（超时、重试策略、错误模型、观测 hook）。

## 当前状态

- `http`、`tcp`、`udp` 子包已具备基线能力。
- 共享重试/TLS 配置原语已存在。
- 主要缺口：跨协议可观测性 hook、弹性策略一致化。

## 版本规划（建议）

- `v0.3`：稳定首批协议基线（`http/tcp/udp`）与共享工程约束
- `v0.4`：typed error 模型 + observability hook
- `v0.5`：可插拔弹性策略（backoff/jitter/circuit-breaker）

## 优先级建议

### P0（当前）

- 强化 `udp` 行为与测试，提升到可稳定复用水平。
- 定义共享约束（timeout、retry、错误分类、hook 生命周期），不定义统一调用 API。
- 修复协议层连接生命周期边界问题并补齐测试。

### P1（下一阶段）

- 完成 typed error 在三协议路径的覆盖与兼容性测试。
- 增加与 `observabilityx` 对齐的可观测性 hook。
- 对齐三协议配置语义，降低切换成本。

### P2（后续）

- 增加可插拔弹性策略（backoff/jitter/circuit-breaker）。
- 提供 transport 扩展点，同时保持核心轻量。

## 非目标

- 不做完全抹平协议语义的一刀切抽象。
- 不替代成熟协议 SDK 的全部能力。
- 不强绑单一遥测或重试实现。

## 迁移来源

- 内容汇总自 ArcGo 全局 roadmap 草案与当前包状态。
- 本页为 docs 内维护的正式 roadmap。
