---
title: 'ID Generation'
linkTitle: 'ID Generation'
description: 'dbx 中的强类型 ID 生成策略'
weight: 9
---

## ID 生成

`dbx` 通过 `NewIDColumn[..., ..., Marker]()` 提供主键 ID 的强类型策略配置。

## Marker 类型

| Marker | ID 类型 | 行为 |
| --- | --- | --- |
| `dbx.IDAuto` | `int64` | 数据库自增 / identity |
| `dbx.IDSnowflake` | `int64` | 应用侧生成 Snowflake ID |
| `dbx.IDUUID` | `string` | 应用侧生成 UUID（默认 v7） |
| `dbx.IDUUIDv7` | `string` | 应用侧生成 UUIDv7 |
| `dbx.IDUUIDv4` | `string` | 应用侧生成 UUIDv4 |

## 示例

```go
ID: dbx.NewIDColumn[Event, int64, dbx.IDSnowflake](),
```

## 默认规则

- `int64` 主键默认 `db_auto`
- `string` 主键默认 `uuid(v7)`

## 迁移说明

`idgen` / `uuidv` 标签参数已移除，请使用 marker type 配置。
