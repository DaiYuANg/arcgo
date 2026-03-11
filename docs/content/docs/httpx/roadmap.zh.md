---
title: 'roadmap'
linkTitle: 'roadmap'
description: 'httpx 路线图'
weight: 90
---

## httpx Roadmap（2026-03）

## 定位

`httpx` 是基于 Huma 的统一 HTTP 服务组织层，不是重型框架。

- 提供一致的 server/group/endpoint API
- 保留 Huma 高级能力直连出口
- 支持 adapter 原生生态与 Huma 语义层并存

## 当前状态

- 核心 API 面基本成形（OpenAPI/docs/security/group 能力已落地）
- 架构收敛已完成一轮（配置职责从 `httpx` 回收到 adapter）
- 主要缺口：adapter middleware 正式 API、adapter 构造期配置文档与一致性

## 优先级建议

### P0（立即）

- 完成各 adapter 构造期 `Options` 收口（logger/timeout/shutdown）
- 补齐这部分的单元测试和示例
- 补文档：明确 `httpx` 层日志、adapter bridge 日志、框架原生日志边界

## P0 执行记录（2026-03-11）

- Huma 版本已从 `v2.36.0` 升级到 `v2.37.2`
- 回归验证通过：
  - `go test ./httpx/...`
  - `go test ./...`
- 已补齐 adapter 构造期 `Options` 测试（`std`/`gin`/`echo`/`fiber`）：
  - 覆盖 logger 注入与错误日志路径
  - 覆盖 timeout/shutdown 默认值与覆盖值合并行为
- 已更新示例以展示构造期 `Options` 用法：
  - `httpx/examples/std`
  - `httpx/examples/gin`
  - `httpx/examples/echo`
  - `httpx/examples/fiber`
- 兼容性清单（已核对）：
  - 类型化路由注册（`huma.Register`）行为保持兼容
  - Group 相关能力（middleware/modifier/transformer）行为保持兼容
  - OpenAPI/doc 路径配置与运行时 `ConfigureDocs(...)` 重绑定行为保持兼容
  - Security / Components / Global 参数相关 OpenAPI patch 行为保持兼容
  - `std`/`gin`/`echo`/`fiber` adapter 的 Huma docs 控制器行为保持兼容
- 仍需后续跟进：
  - `fiber` adapter 不支持 `net/http` `ServeHTTP`（当前设计返回 `501`）
  - `httpx/fx` 仍是轻量封装，缺少生命周期集成测试

### P1（下一阶段）

- 落地 `UseAdapterMiddleware(...)` 或同级正式入口
- 强化 group/endpoint 默认能力收口（避免零散 helper）
- 完整文档化 docs renderer 与 OpenAPI patch 组合用法

### P2（后续）

- 针对性能敏感路径做基准与回归守护
- 对常见组织模式给出模板化示例（auth/org/observability）

## 非目标

- 不替代 Huma
- 不把 adapter native middleware 与 Huma middleware 强行混成一种机制
- 不引入重型 runtime/framework 生命周期系统

## 调整说明

相对历史 roadmap，`httpx` 当前应以“API 收敛 + 配置一致性”优先，
暂不建议继续扩展大量新的 helper 面，否则会再次引入语义漂移。

## 迁移来源

- 历史包内文件（已删除）：`httpx/ROADMAP.md`
- 本页为 docs 内维护的正式版本
