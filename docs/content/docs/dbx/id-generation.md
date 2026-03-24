---
title: 'ID Generation'
linkTitle: 'ID Generation'
description: 'Typed ID generation strategies in dbx'
weight: 9
---

## ID Generation

`dbx` supports typed ID generation strategies for primary keys.  
Configure ID behavior directly in schema fields with `IDColumn[..., ..., Marker]`, not string tags.

## Marker Types

| Marker | ID type | Behavior |
| --- | --- | --- |
| `dbx.IDAuto` | `int64` | Database auto-increment/identity (default for `int64` PK) |
| `dbx.IDSnowflake` | `int64` | App-generated Snowflake ID |
| `dbx.IDUUID` | `string` | App-generated UUID (defaults to v7) |
| `dbx.IDUUIDv7` | `string` | App-generated UUIDv7 |
| `dbx.IDUUIDv4` | `string` | App-generated UUIDv4 |
| `dbx.IDULID` | `string` | App-generated ULID |
| `dbx.IDKSUID` | `string` | App-generated KSUID |

## Recommended Usage

```go
type Event struct {
    ID   int64  `dbx:"id"`
    Name string `dbx:"name"`
}

type EventSchema struct {
    dbx.Schema[Event]
    ID   dbx.IDColumn[Event, int64, dbx.IDSnowflake] `dbx:"id,pk"`
    Name dbx.Column[Event, string]                   `dbx:"name"`
}
var Events = dbx.MustSchema("events", EventSchema{})
```

## Defaults

- `int64` primary key with `dbx:"id,pk"` defaults to `db_auto`.
- `string` primary key with `dbx:"id,pk"` defaults to `uuid` with version `v7`.

## Production Guidance

- Single-instance can rely on the default node id behavior.
- Multi-instance should set an explicit stable node id with `dbx.WithNodeID(...)`.
- Keep declaration and execution separated: schema uses `IDColumn`, runtime generation uses DB options.
- `WithNodeID` and `WithIDGenerator` are mutually exclusive; configuring both returns an error.

## Migration Note

`idgen` and `uuidv` tag parameters are removed.  
Use marker types on `IDColumn` for explicit ID strategy configuration.
