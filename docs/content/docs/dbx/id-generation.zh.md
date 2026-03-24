---
title: 'ID Generation'
linkTitle: 'ID Generation'
description: 'dbx 中的强类型 ID 生成策略'
weight: 9
---

## ID 生成

`dbx` 通过 `IDColumn[..., ..., Marker]` 在 schema 字段声明中直接配置主键 ID 策略。

## Marker 类型

| Marker | ID 类型 | 行为 |
| --- | --- | --- |
| `dbx.IDAuto` | `int64` | 数据库自增 / identity |
| `dbx.IDSnowflake` | `int64` | 应用侧生成 Snowflake ID |
| `dbx.IDUUID` | `string` | 应用侧生成 UUID（默认 v7） |
| `dbx.IDUUIDv7` | `string` | 应用侧生成 UUIDv7 |
| `dbx.IDUUIDv4` | `string` | 应用侧生成 UUIDv4 |
| `dbx.IDULID` | `string` | 应用侧生成 ULID |
| `dbx.IDKSUID` | `string` | 应用侧生成 KSUID |

## 示例

```go
type EventSchema struct {
    dbx.Schema[Event]
    ID   dbx.IDColumn[Event, int64, dbx.IDSnowflake] `dbx:"id,pk"`
    Name dbx.Column[Event, string]                   `dbx:"name"`
}
```

## 默认规则

- `int64` 主键默认 `db_auto`
- `string` 主键默认 `uuid(v7)`

## 生产建议

- 单实例场景可以使用默认 node id 行为。
- 多实例场景建议通过 `dbx.WithNodeID(...)` 显式配置稳定 node id。
- 保持分层：schema 用 `IDColumn` 声明策略，运行时通过 DB option 配置生成器。
- `WithNodeID` 与 `WithIDGenerator` 互斥，同时配置会返回错误。

## 迁移说明

`idgen` / `uuidv` 标签参数已移除，请在 `IDColumn` 上使用 marker type 配置。
