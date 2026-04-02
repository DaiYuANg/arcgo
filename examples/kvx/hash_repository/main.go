// Package main demonstrates kvx hash repository usage with the in-memory demo backend.
package main

import (
	"context"
	"fmt"

	"github.com/DaiYuANg/arcgo/examples/kvx/shared"
	"github.com/DaiYuANg/arcgo/kvx/mapping"
	"github.com/DaiYuANg/arcgo/kvx/repository"
)

func main() {
	ctx := context.Background()
	backend := shared.NewHashBackend()

	preset := repository.NewPreset[shared.User](
		repository.WithKeyBuilder[shared.User](mapping.NewKeyBuilder("demo:user")),
	)

	repo := repository.NewHashRepository[shared.User](backend, backend, "user", preset.HashOptions(
		repository.WithHashCodec[shared.User](mapping.NewHashCodec(nil)),
	)...)

	must(repo.Save(ctx, &shared.User{ID: "u-1", Name: "Alice", Email: "alice@example.com"}))
	must(repo.Save(ctx, &shared.User{ID: "u-2", Name: "Bob", Email: "bob@example.com"}))

	entity, err := repo.FindByID(ctx, "u-1")
	must(err)

	matches, err := repo.FindByField(ctx, "email", "alice@example.com")
	must(err)

	count, err := repo.Count(ctx)
	must(err)

	mustPrintf("loaded: %s (%s)\n", entity.Name, entity.Email)
	mustPrintf("indexed matches: %d\n", matches.Len())
	mustPrintf("count: %d\n", count)
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
