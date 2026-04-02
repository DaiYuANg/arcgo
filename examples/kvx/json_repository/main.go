// Package main demonstrates kvx JSON repository usage with the in-memory demo backend.
package main

import (
	"context"
	"fmt"

	"github.com/DaiYuANg/arcgo/examples/kvx/shared"
	"github.com/DaiYuANg/arcgo/kvx/repository"
)

func main() {
	ctx := context.Background()
	backend := shared.NewJSONBackend()
	repo := repository.NewJSONRepository[shared.User](backend, backend, "json:user")

	must(repo.Save(ctx, &shared.User{ID: "u-1", Name: "Alice", Email: "alice@example.com"}))
	must(repo.Save(ctx, &shared.User{ID: "u-2", Name: "Bob", Email: "bob@example.com"}))

	exists, err := repo.Exists(ctx, "u-1")
	must(err)

	entity, err := repo.FindByID(ctx, "u-2")
	must(err)

	must(repo.UpdateField(ctx, "u-2", "$.name", "Bobby"))

	updated, err := repo.FindByID(ctx, "u-2")
	must(err)

	all, err := repo.FindAll(ctx)
	must(err)

	mustPrintf("exists u-1: %v\n", exists)
	mustPrintf("loaded: %s (%s)\n", entity.ID, entity.Email)
	mustPrintf("updated name: %s\n", updated.Name)
	mustPrintf("total: %d\n", all.Len())
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func mustPrintf(format string, args ...any) {
	if _, err := fmt.Printf(format, args...); err != nil {
		panic(err)
	}
}
