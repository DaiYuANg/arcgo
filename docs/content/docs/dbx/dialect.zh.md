---
title: 'dbx Dialect'
linkTitle: 'Dialect'
description: 'dbx 与 sqltmplx 的方言抽象'
weight: 11
---

## dialect

dbx 与 sqltmplx 的方言抽象。能力分层，按需实现即可。

## Capability layers 能力层

| 层级 | 接口 | 是否必须 | 使用方 |
|------|------|----------|--------|
| **Contract** | `Name()`, `BindVar(n)` | 是 | sqltmplx render、dbx render、validate |
| **Dialect** | Contract + `QuoteIdent`, `RenderLimitOffset` | query DSL 必须 | dbx query 构建 |
| **QueryFeaturesProvider** | `QueryFeatures()` | 可选 | dbx render（upsert、RETURNING、excluded ref）。已知方言可回退到 `DefaultQueryFeatures(name)` |
| **SchemaDialect** | Dialect + DDL/inspect（在 dbx） | 可选 | schema migrate、AutoMigrate |

## Adding a new dialect 新增方言

1. 实现 `dialect.Dialect`（Contract + QuoteIdent + RenderLimitOffset）。
2. 实现 `dialect.QueryFeaturesProvider` 声明 upsert/returning 支持（或使用 `DefaultQueryFeatures` 若方言匹配已知类型）。
3. 若需 schema migration：实现 `dbx.SchemaDialect`（BuildCreateTable、InspectTable 等），或使用 Atlas 支持时依赖 Atlas。
4. 若需 sqltmplx 校验：通过 `validate.Register(dialectName, factory)` 注册 parser。

无需在 render.go、schema_migrate_atlas.go 或 sqltmplx 中增加方言分支——能力通过接口声明即可。
