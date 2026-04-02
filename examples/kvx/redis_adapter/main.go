// Package redis_adapter demonstrates using the kvx Redis adapter directly.
package main

import (
	"context"
	"fmt"

	"github.com/DaiYuANg/arcgo/examples/kvx/shared"
	"github.com/DaiYuANg/arcgo/kvx"
	redisadapter "github.com/DaiYuANg/arcgo/kvx/adapter/redis"
	"github.com/DaiYuANg/arcgo/kvx/repository"
)

func main() {
	ctx := context.Background()

	container, addr, err := shared.StartContainer(ctx, shared.RedisImage())
	must(err)
	defer func() { must(container.Terminate(ctx)) }()

	adapter, err := redisadapter.New(kvx.ClientOptions{
		Addrs: []string{addr},
	})
	must(err)
	defer func() { must(adapter.Close()) }()

	repo := repository.NewHashRepository[shared.User](
		adapter,
		adapter,
		"demo:user",
		repository.WithPipeline[shared.User](adapter),
	)

	must(repo.Save(ctx, &shared.User{
		ID:    "u-1",
		Name:  "Alice",
		Email: "alice@example.com",
	}))
	must(repo.Save(ctx, &shared.User{
		ID:    "u-2",
		Name:  "Bob",
		Email: "bob@example.com",
	}))

	entity, err := repo.FindByID(ctx, "u-1")
	must(err)

	matches, err := repo.FindByField(ctx, "email", "alice@example.com")
	must(err)

	count, err := repo.Count(ctx)
	must(err)

	mustWritef("redis addr: %s\n", addr)
	mustWritef("loaded: %s (%s)\n", entity.Name, entity.Email)
	mustWritef("indexed matches: %d\n", matches.Len())
	mustWritef("count: %d\n", count)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func mustWritef(format string, args ...any) {
	_, err := fmt.Printf(format, args...)
	must(err)
}
