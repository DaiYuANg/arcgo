---
title: 'kvx'
linkTitle: 'kvx'
description: 'Strongly Typed Redis / Valkey Object Access and Repository Layer'
weight: 6
---

## kvx

`kvx` is a layered Redis / Valkey access package focused on strongly typed object access, repository-style persistence, and Redis-native capabilities.

## What You Get

- Unified `Client` capability interfaces for `KV`, `Hash`, `JSON`, `PubSub`, `Stream`, `Search`, `Script`, and `Lock`
- Metadata-driven mapping based on `kvx` struct tags
- `HashRepository` and `JSONRepository` for strongly typed persistence
- Secondary-index helper support through repository indexers
- Feature modules for `json`, `pubsub`, `stream`, `search`, and `lock`
- Thin adapters for Redis and Valkey drivers

## Positioning

`kvx` is not trying to be a generic cache abstraction.
It is a Redis / Valkey-oriented object access layer for services that want typed repositories without giving up Redis-native data models.

## Minimal Repository Example

```go
type User struct {
    ID    string `kvx:"id"`
    Name  string `kvx:"name"`
    Email string `kvx:"email,index=email"`
}

backend := shared.NewHashBackend()
repo := repository.NewHashRepository[User](backend, backend, "user")

_ = repo.Save(ctx, &User{
    ID:    "u-1",
    Name:  "Alice",
    Email: "alice@example.com",
})

entity, _ := repo.FindByID(ctx, "u-1")
matches, _ := repo.FindByField(ctx, "email", "alice@example.com")
_, _ = entity, matches
```

## Core Layers

### Client Interfaces

- `KV`
- `Hash`
- `JSON`
- `PubSub`
- `Stream`
- `Search`
- `Script`
- `Lock`
- `Client`

### Mapping

`kvx` struct tags drive schema metadata:

```go
type User struct {
    ID    string `kvx:"id"`
    Name  string `kvx:"name"`
    Email string `kvx:"email,index=email"`
}
```

Supported metadata concepts include:

- key field
- storage field name
- indexed field
- custom index alias

### Repositories

- `repository.NewHashRepository[T](...)`
- `repository.NewJSONRepository[T](...)`
- `repository.NewPreset[T](...)`
- `repository.WithKeyBuilder(...)`
- `repository.WithIndexer(...)`
- `repository.WithHashCodec(...)`
- `repository.WithSerializer(...)`

## Feature Modules

- `module/json`: higher-level JSON document helpers
- `module/pubsub`: channel subscription management
- `module/stream`: stream and consumer-group helpers
- `module/search`: RediSearch-oriented query helpers
- `module/lock`: distributed lock helpers

## Adapters

- `kvx/adapter/redis`
- `kvx/adapter/valkey`

These adapters stay thin and primarily expose the `kvx` capability surface over the underlying driver.

## Examples

- `go run ./examples/kvx/hash_repository`
  - in-memory hash repository flow with indexing
- `go run ./examples/kvx/json_repository`
  - in-memory JSON repository flow with field updates and scanning
- `go run ./examples/kvx/redis_adapter`
  - real Redis-backed hash repository flow using `testcontainers-go`
- `go run ./examples/kvx/redis_hash`
  - real Redis hash example using `testcontainers-go`
- `go run ./examples/kvx/redis_json`
  - real Redis JSON example using `testcontainers-go`
- `go run ./examples/kvx/redis_stream`
  - real Redis stream example using `testcontainers-go`
- `go run ./examples/kvx/valkey_hash`
  - real Valkey hash example using `testcontainers-go`
- `go run ./examples/kvx/valkey_json`
  - real Valkey JSON example using `testcontainers-go`
- `go run ./examples/kvx/valkey_stream`
  - real Valkey stream example using `testcontainers-go`

## Container Images

- `redis_hash` and `redis_stream` default to `redis:7-alpine`
- `redis_json` defaults to `redis/redis-stack-server:latest`
- `valkey_hash` and `valkey_stream` default to `valkey/valkey:8-alpine`
- `valkey_json` defaults to `valkey/valkey:8-alpine`; override `KVX_VALKEY_JSON_IMAGE` if your JSON commands require a different image

## Notes

- The repository layer is currently the most mature part of `kvx`.
- `FindAll` / `Count` now scan the full keyspace cursor path instead of only a single page.
- Workspace sibling modules like `collectionx` resolve through `go.work`; no extra local dependency declaration is needed in `kvx/go.mod`.