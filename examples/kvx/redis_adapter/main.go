package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/DaiYuANg/arcgo/kvx"
	redisadapter "github.com/DaiYuANg/arcgo/kvx/adapter/redis"
	"github.com/DaiYuANg/arcgo/kvx/repository"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type User struct {
	ID    string `kvx:"id"`
	Name  string `kvx:"name"`
	Email string `kvx:"email,index=email"`
}

func main() {
	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	must(err)
	defer func() {
		_ = container.Terminate(ctx)
	}()

	host, err := container.Host(ctx)
	must(err)

	port, err := container.MappedPort(ctx, "6379/tcp")
	must(err)

	adapter, err := redisadapter.New(kvx.ClientOptions{
		Addrs: []string{net.JoinHostPort(host, port.Port())},
	})
	must(err)
	defer func() {
		_ = adapter.Close()
	}()

	repo := repository.NewHashRepository[User](
		adapter,
		adapter,
		"demo:user",
		repository.WithPipeline[User](adapter),
	)

	must(repo.Save(ctx, &User{
		ID:    "u-1",
		Name:  "Alice",
		Email: "alice@example.com",
	}))
	must(repo.Save(ctx, &User{
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

	fmt.Printf("redis addr: %s\n", net.JoinHostPort(host, port.Port()))
	fmt.Printf("loaded: %s (%s)\n", entity.Name, entity.Email)
	fmt.Printf("indexed matches: %d\n", len(matches))
	fmt.Printf("count: %d\n", count)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
