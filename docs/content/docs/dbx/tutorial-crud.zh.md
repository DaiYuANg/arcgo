---
title: 'CRUD 教程'
linkTitle: 'tutorial-crud'
description: '使用 dbx 完成端到端 CRUD 的完整示例'
weight: 13
---

## CRUD 教程

本页展示 `dbx` 的完整 CRUD 流程：建表、插入、查询、更新、删除。

## 可运行完整示例

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
	raw, err := sql.Open("sqlite3", "file:dbx_crud.db?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	defer raw.Close()

	core, err := dbx.NewWithOptions(raw, sqlite.New(), dbx.WithDebug(true))
	if err != nil {
		log.Fatal(err)
	}
	if err := core.AutoMigrate(ctx, Users); err != nil {
		log.Fatal(err)
	}

	mapper := dbx.MustMapper[User](Users)
	u := &User{Username: "alice", Email: "alice@example.com", Status: 1}
	assignments, err := mapper.InsertAssignments(core, Users, u)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := dbx.Exec(ctx, core, dbx.InsertInto(Users).Values(assignments...)); err != nil {
		log.Fatal(err)
	}

	list, err := dbx.QueryAll(
		ctx, core,
		dbx.Select(Users.AllColumns().Values()...).From(Users).Where(Users.Username.Eq("alice")),
		mapper,
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("query rows=%d\n", len(list))

	if _, err := dbx.Exec(
		ctx, core,
		dbx.Update(Users).Set(Users.Status.Assign(2)).Where(Users.Username.Eq("alice")),
	); err != nil {
		log.Fatal(err)
	}

	if _, err := dbx.Exec(ctx, core, dbx.DeleteFrom(Users).Where(Users.Username.Eq("alice"))); err != nil {
		log.Fatal(err)
	}
}
```
