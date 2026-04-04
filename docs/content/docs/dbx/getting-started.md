---
title: 'dbx Getting Started'
linkTitle: 'getting-started'
description: 'Build and run your first dbx app'
weight: 7
---

## Getting Started

This guide shows a complete, runnable dbx example from schema definition to query execution.

## When to Use

- You are starting a new service with `database/sql`.
- You want schema-first metadata with typed query APIs.
- You want one minimal runnable sample as baseline.

## Minimal Project Layout

```text
.
├── go.mod
└── main.go
```

## 1) Install Dependencies

```bash
go get github.com/DaiYuANg/arcgo/dbx
go get github.com/DaiYuANg/arcgo/dbx/dialect/sqlite
go get github.com/mattn/go-sqlite3
```

## 2) Create `main.go`

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/DaiYuANg/arcgo/dbx/dialect/sqlite"

	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	ID       int64  `dbx:"id"`
	Username string `dbx:"username"`
	Email    string `dbx:"email"`
	Status   int    `dbx:"status"`
}

type UserSchema struct {
	dbx.Schema[User]
	ID       dbx.IDColumn[User, int64, dbx.IDSnowflake] `dbx:"id,pk"`
	Username dbx.Column[User, string]                   `dbx:"username,index"`
	Email    dbx.Column[User, string]                   `dbx:"email,unique"`
	Status   dbx.Column[User, int]                      `dbx:"status,default=1,index"`
}

var Users = dbx.MustSchema("users", UserSchema{})

func main() {
	ctx := context.Background()

	raw, err := sql.Open("sqlite3", "file:dbx_getting_started.db?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	defer raw.Close()

	core, err := dbx.NewWithOptions(
		raw,
		sqlite.New(),
		dbx.WithDebug(true),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Create/align table structure based on schema metadata.
	if err := core.AutoMigrate(ctx, Users); err != nil {
		log.Fatal(err)
	}

	mapper := dbx.MustMapper[User](Users)
	alice := &User{
		Username: "alice",
		Email:    "alice@example.com",
		Status:   1,
	}

	assignments, err := mapper.InsertAssignments(core, Users, alice)
	if err != nil {
		log.Fatal(err)
	}

	if _, err := dbx.Exec(ctx, core, dbx.InsertInto(Users).Values(assignments...)); err != nil {
		log.Fatal(err)
	}

	items, err := dbx.QueryAll(
		ctx,
		core,
		dbx.Select(Users.AllColumns().Values()...).From(Users).Where(Users.Status.Eq(1)),
		mapper,
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("active users: %d\n", len(items))
	for _, item := range items {
		fmt.Printf("id=%d username=%s email=%s status=%d\n", item.ID, item.Username, item.Email, item.Status)
	}
}
```

## 3) Run

```bash
go run .
```

## Pitfalls

- Forgetting `AutoMigrate` before first write often causes "no such table" errors.
- Mixing schema metadata across multiple structs for one table creates confusion; keep one schema source.
- Using `dbx.WithNodeID` and `dbx.WithIDGenerator` together is invalid.

## Verify

```bash
go test ./dbx/...
go run .
```

## Next Steps

- ID strategy and production guidance: [ID Generation](./id-generation)
- Runtime options: [Options](./options)
- Logging and hooks: [Observability](./observability)
- Full runnable examples: [Examples](./examples)
