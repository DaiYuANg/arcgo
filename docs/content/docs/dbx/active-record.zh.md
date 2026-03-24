---
title: 'Active Record 模式'
linkTitle: 'active-record'
description: '基于 dbx repository.Base 的轻量 Active Record 门面'
weight: 20
---

## Active Record 模式

包路径：`github.com/DaiYuANg/arcgo/dbx/activerecord`。

`activerecord` 构建在 `github.com/DaiYuANg/arcgo/dbx/repository` 之上的一层薄封装：把实体包在 `Model` 里，读写仍走与 Repository 模式相同的 `repository.Base`，**没有第二套查询引擎**。

## 适用场景

- 希望在实例上直接调用 `Save`、`Reload`、`Delete` 这类面向对象的 API。
- 仍要保留 schema-first 类型安全，并偶尔需要批量查询、事务等能力时，通过 `Store.Repository()` 使用完整仓储 API。

## `Store` 与 `Model`

- `activerecord.New[E](db *dbx.DB, schema S) *Store[E, S]` — 内部持有 `*repository.Base[E, S]`。
- `Store.Repository() *repository.Base[E, S]` — 逃生舱：批量操作、Spec、事务等仍用仓储层。
- `Store.Wrap(entity *E) *Model[E, S]` — 把实体指针挂到当前 `Store`。
- `Store.FindByID`、`Store.FindByKey`、`Store.List` — 返回 `*Model`；查不到行时错误链上会出现 `repository.ErrNotFound`（与仓储一致）。
- `Model.Entity() *E`、`Model.Key() repository.Key` — `Key` 为当前主键的**防御性拷贝**（`map[string]any`）。
- `Model.Save` — 主键为空或各主键字段为零值时执行插入；否则按主键更新。若按主键更新影响行数为 0，会回退为插入（用于「行已不存在」一类场景）。
- `Model.Reload`、`Model.Delete` — 均按当前 `Key` 与仓储交互。

## 可选查询（`mo.Option`）

与仓储层 `GetByIDOption` / `GetByKeyOption` 对齐的并行 API：

- `Store.FindByIDOption(ctx, id) (mo.Option[*Model[E, S]], error)`
- `Store.FindByKeyOption(ctx, key) (mo.Option[*Model[E, S]], error)`

当记录不存在时，返回 `mo.None[*Model[E, S]]()` 且 **`error` 为 `nil`**，与 `repository.GetByIDOption`、`GetByKeyOption` 语义一致。数据库错误、校验错误等仍为**非 nil `error`**。

## 完整示例

```go
package main

import (
	"context"
	"database/sql"

	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/DaiYuANg/arcgo/dbx/activerecord"
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
	Name dbx.Column[User, string] `dbx:"name"`
}

var Users = dbx.MustSchema("users", UserSchema{})

func main() {
	ctx := context.Background()
	raw, _ := sql.Open("sqlite3", "file:ar_example.db?cache=shared")
	core := dbx.MustNewWithOptions(raw, sqlite.New())
	_, _ = core.AutoMigrate(ctx, Users)

	store := activerecord.New[User](core, Users)
	m := store.Wrap(&User{Name: "alice"})
	_ = m.Save(ctx)

	opt, err := store.FindByIDOption(ctx, m.Entity().ID)
	if err != nil {
		return
	}
	_, _ = opt.Get()

	_, _ = store.Repository().ListSpec(ctx, repository.Where(Users.Name.Eq("alice")))
}
```

`FindByIDOption` 的返回类型为 `mo.Option[*Model[User, UserSchema]]`（包 `github.com/samber/mo`）；在业务代码里若需显式构造 `mo.Some` / `mo.None`，再添加对应 import 即可。

## 相关文档

- [Repository 模式](./repository) — 底层 `repository.Base`、Spec、错误模型与 `mo.Option` 读接口。
- 英文版对照：[Active Record Mode](./active-record)（与本文内容一致，便于检索）。
