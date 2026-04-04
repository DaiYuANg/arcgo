---
title: 'dbx 快速开始'
linkTitle: 'getting-started'
description: '从零构建并运行第一个 dbx 程序'
weight: 7
---

## 快速开始

本页提供一个可直接运行的完整示例：从 schema 定义到数据写入、查询。

## 1）安装依赖

```bash
go get github.com/DaiYuANg/arcgo/dbx
go get github.com/DaiYuANg/arcgo/dbx/dialect/sqlite
go get github.com/mattn/go-sqlite3
```

## 2）创建 `main.go`

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

	// 按 schema 元数据创建/对齐表结构。
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

## 3）运行

```bash
go run .
```

## 下一步

- ID 策略与生产建议：[ID Generation](./id-generation)
- 运行时配置：[Options](./options)
- 日志与 Hook：[Observability](./observability)
- 完整可运行示例：[Examples](./examples)
