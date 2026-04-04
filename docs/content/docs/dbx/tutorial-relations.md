---
title: 'Relations Tutorial'
linkTitle: 'tutorial-relations'
description: 'BelongsTo and batch relation loading in dbx'
weight: 14
---

## Relations Tutorial

This tutorial shows relation declaration and batch loading with `LoadBelongsTo`.

## When to Use

- You have normalized tables and want typed relation metadata.
- You want to avoid N+1 query patterns with batch relation loading.
- You want attach callbacks per source entity.

## Minimal Project Layout

```text
.
├── go.mod
└── main.go
```

## Complete Runnable Example

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/DaiYuANg/arcgo/dbx/dialect/sqlite"
	"github.com/samber/mo"

	_ "github.com/mattn/go-sqlite3"
)

type Role struct {
	ID   int64  `dbx:"id"`
	Name string `dbx:"name"`
}

type User struct {
	ID       int64  `dbx:"id"`
	RoleID   int64  `dbx:"role_id"`
	Username string `dbx:"username"`
}

type RoleSchema struct {
	dbx.Schema[Role]
	ID   dbx.IDColumn[Role, int64, dbx.IDSnowflake] `dbx:"id,pk"`
	Name dbx.Column[Role, string]                   `dbx:"name,unique"`
}

type UserSchema struct {
	dbx.Schema[User]
	ID       dbx.IDColumn[User, int64, dbx.IDSnowflake] `dbx:"id,pk"`
	RoleID   dbx.Column[User, int64]                    `dbx:"role_id,ref=roles.id,ondelete=cascade,index"`
	Username dbx.Column[User, string]                   `dbx:"username,index"`
	Role     dbx.BelongsTo[User, Role]                  `rel:"table=roles,local=role_id,target=id"`
}

var Roles = dbx.MustSchema("roles", RoleSchema{})
var Users = dbx.MustSchema("users", UserSchema{})

func main() {
	ctx := context.Background()
	raw, err := sql.Open("sqlite3", "file:dbx_relations.db?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	defer raw.Close()

	core, err := dbx.NewWithOptions(raw, sqlite.New())
	if err != nil {
		log.Fatal(err)
	}
	if err := core.AutoMigrate(ctx, Roles, Users); err != nil {
		log.Fatal(err)
	}

	userMapper := dbx.MustMapper[User](Users)
	roleMapper := dbx.MustMapper[Role](Roles)

	items, err := dbx.QueryAll(ctx, core, dbx.Select(Users.AllColumns().Values()...).From(Users), userMapper)
	if err != nil {
		log.Fatal(err)
	}

	if err := dbx.LoadBelongsTo(
		ctx,
		core,
		items,
		Users,
		userMapper,
		Users.Role,
		Roles,
		roleMapper,
		func(index int, user *User, role mo.Option[Role]) {
			if role.IsPresent() {
				value, _ := role.Get()
				fmt.Printf("user=%s role=%s\n", user.Username, value.Name)
			}
		},
	); err != nil {
		log.Fatal(err)
	}
}
```

## Pitfalls

- Missing `rel` tag fields (`table`, `local`, `target`) breaks relation loading.
- Source key type and target key type must be compatible.
- Forgetting to migrate both source and target schemas causes runtime query errors.

## Verify

```bash
go test ./dbx/... -run Relation
go run .
```
