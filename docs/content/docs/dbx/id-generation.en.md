---
title: 'ID Generation'
linkTitle: 'ID Generation'
description: 'Typed ID generation strategies in dbx'
weight: 9
---

## ID Generation

Use `NewIDColumn[..., ..., Marker]()` to configure primary-key ID generation with typed marker strategies.

## Marker Types

| Marker | ID type | Behavior |
| --- | --- | --- |
| `dbx.IDAuto` | `int64` | Database auto-increment/identity |
| `dbx.IDSnowflake` | `int64` | App-generated Snowflake ID |
| `dbx.IDUUID` | `string` | App-generated UUID (default v7) |
| `dbx.IDUUIDv7` | `string` | App-generated UUIDv7 |
| `dbx.IDUUIDv4` | `string` | App-generated UUIDv4 |

## Example

```go
ID: dbx.NewIDColumn[Event, int64, dbx.IDSnowflake](),
```

## Defaults

- `int64` primary key defaults to `db_auto`
- `string` primary key defaults to `uuid` (`v7`)

## Migration Note

`idgen` / `uuidv` tags are removed. Use marker types instead.
