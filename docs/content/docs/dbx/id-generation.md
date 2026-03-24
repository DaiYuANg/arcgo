---
title: 'ID Generation'
linkTitle: 'ID Generation'
description: 'Typed ID generation strategies in dbx'
weight: 9
---

## ID Generation

`dbx` supports typed ID generation strategies for primary keys.  
Configure ID behavior with `NewIDColumn[..., ..., Marker]()` and marker types, not string tags.

## Marker Types

| Marker | ID type | Behavior |
| --- | --- | --- |
| `dbx.IDAuto` | `int64` | Database auto-increment/identity (default for `int64` PK) |
| `dbx.IDSnowflake` | `int64` | App-generated Snowflake ID |
| `dbx.IDUUID` | `string` | App-generated UUID (defaults to v7) |
| `dbx.IDUUIDv7` | `string` | App-generated UUIDv7 |
| `dbx.IDUUIDv4` | `string` | App-generated UUIDv4 |

## Recommended Usage

```go
type Event struct {
    ID   int64  `dbx:"id"`
    Name string `dbx:"name"`
}

type EventSchema struct {
    dbx.Schema[Event]
    ID   dbx.Column[Event, int64]  `dbx:"id,pk"`
    Name dbx.Column[Event, string] `dbx:"name"`
}

var Events = dbx.MustSchema("events", EventSchema{
    ID: dbx.NewIDColumn[Event, int64, dbx.IDSnowflake](),
})
```

## Defaults

- `int64` primary key with `dbx:"id,pk"` defaults to `db_auto`.
- `string` primary key with `dbx:"id,pk"` defaults to `uuid` with version `v7`.

## Migration Note

`idgen` and `uuidv` tag parameters are removed.  
Use marker types (`NewIDColumn`) for explicit ID strategy configuration.
