---
title: 'Repository Mode'
linkTitle: 'repository'
description: 'Generic repository abstraction on top of dbx schema-first core'
weight: 19
---

## Repository Mode

`dbx/repository` is a thin abstraction over `dbx` core APIs. It keeps schema-first typing while offering service-friendly CRUD methods.

## When to Use

- You prefer explicit domain repositories over active-record style methods.
- You want transaction boundaries and query behavior centralized per aggregate.
- You need reusable CRUD, pagination, upsert, and spec-based filtering.

## Complete Example

```go
package main

import (
	"context"
	"database/sql"

	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/DaiYuANg/arcgo/dbx/dialect/sqlite"
	"github.com/DaiYuANg/arcgo/dbx/repository"
)

type User struct {
	ID   int64  `dbx:"id"`
	Name string `dbx:"name"`
}

type UserSchema struct {
	dbx.Schema[User]
	ID   dbx.IDColumn[User, int64, dbx.IDSnowflake] `dbx:"id,pk"`
	Name dbx.Column[User, string]                   `dbx:"name,index"`
}

var Users = dbx.MustSchema("users", UserSchema{})

func main() {
	ctx := context.Background()
	raw, _ := sql.Open("sqlite3", "file:repo_example.db?cache=shared")
	core := dbx.MustNewWithOptions(raw, sqlite.New())
	_, _ = core.AutoMigrate(ctx, Users)

	repo := repository.NewWithOptions[User](core, Users, repository.WithByIDNotFoundAsError(true))
	_ = repo.CreateMany(ctx, &User{Name: "alice"}, &User{Name: "bob"})
	_ = repo.Upsert(ctx, &User{ID: 1, Name: "alice-v2"})
	_, _ = repo.ListPageSpec(ctx, 1, 20, repository.Where(Users.Name.Eq("alice-v2")))
}
```

## API Highlights

- CRUD: `Create`, `CreateMany`, `List`, `First`, `Update`, `Delete`
- PK helpers: `GetByID`, `UpdateByID`, `DeleteByID`
- Composite key helpers: `GetByKey`, `UpdateByKey`, `DeleteByKey`
- Pagination: `ListPage`, `ListPageSpec`
- Upsert: `Upsert(ctx, entity, conflictColumns...)`
- Transactions: `InTx`
- Specs: `Where`, `OrderBy`, `Limit`, `Offset`
- Optional single-row reads: `GetByIDOption`, `GetByKeyOption`, `FirstOption`, `FirstSpecOption` (see below)

## Optional reads (`mo.Option`)

For “maybe one row” queries, you can use parallel methods that return `github.com/samber/mo.Option` instead of treating “not found” as `repository.ErrNotFound`:

- `GetByIDOption`, `GetByKeyOption`, `FirstOption`, `FirstSpecOption`

Semantics:

- When the underlying `GetByID` / `GetByKey` / `First` would return `repository.ErrNotFound`, the `Option` variant returns `mo.None[E]()` and **no error** (`error == nil`).
- Any other failure (invalid key, DB error, etc.) still returns a non-nil `error` and `mo.None[E]()`.
- Empty composite `Key{}` still surfaces as `repository.ValidationError` from `GetByKey` / `GetByKeyOption` (not folded into `Option`).

When to use which:

- Prefer `GetByID` / `First` + `errors.Is(err, repository.ErrNotFound)` when not-found is exceptional or you want uniform `if err != nil` handling.
- Prefer `*Option` when absence is a normal outcome and you want `(mo.Option[E], error)` to separate “missing row” from “real errors”, consistent with `dbx.SQLFind` / `dbx.SQLScalarOption`.

```go
// Assumes User, UserSchema, Users, repo, and ctx from the "Complete Example" above.

byID, err := repo.GetByIDOption(ctx, int64(42))
if err != nil {
	return err
}
if user, ok := byID.Get(); ok {
	_ = user.Name
}

byName, err := repo.FirstSpecOption(ctx, repository.Where(Users.Name.Eq("alice")))
if err != nil {
	return err
}
_, _ = byName.Get()
```

## Error Model

- `ErrNotFound`
- `ErrConflict` (`ConflictError`)
- `ErrValidation` (`ValidationError`)
- `ErrVersionConflict` (`VersionConflictError`)

`WithByIDNotFoundAsError(true)` enables strict by-id mutation semantics (`RowsAffected=0 => ErrNotFound`).
