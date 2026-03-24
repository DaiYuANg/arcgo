---
title: 'ID Generation'
linkTitle: 'ID Generation'
description: 'Typed ID generation strategies in dbx'
weight: 9
---

## ID Generation

Use `IDColumn[..., ..., Marker]` to configure primary-key ID generation directly in schema fields.

## Marker Types

| Marker | ID type | Behavior |
| --- | --- | --- |
| `dbx.IDAuto` | `int64` | Database auto-increment/identity |
| `dbx.IDSnowflake` | `int64` | App-generated Snowflake ID |
| `dbx.IDUUID` | `string` | App-generated UUID (default v7) |
| `dbx.IDUUIDv7` | `string` | App-generated UUIDv7 |
| `dbx.IDUUIDv4` | `string` | App-generated UUIDv4 |
| `dbx.IDULID` | `string` | App-generated ULID |
| `dbx.IDKSUID` | `string` | App-generated KSUID |

## Example

```go
type EventSchema struct {
    dbx.Schema[Event]
    ID   dbx.IDColumn[Event, int64, dbx.IDSnowflake] `dbx:"id,pk"`
    Name dbx.Column[Event, string]                   `dbx:"name"`
}
```

## Defaults

- `int64` primary key defaults to `db_auto`
- `string` primary key defaults to `uuid` (`v7`)

## Production Guidance

- Single-instance deployments can use the default node id behavior.
- Multi-instance deployments should configure a stable explicit node id via `dbx.WithNodeID(...)`.
- Configure runtime generation in DB options (`WithNodeID` or `WithIDGenerator`), not in schema.
- `WithNodeID` and `WithIDGenerator` are mutually exclusive. Passing both returns an error.

## Migration Note

`idgen` / `uuidv` tags are removed. Use marker types on `IDColumn`.
