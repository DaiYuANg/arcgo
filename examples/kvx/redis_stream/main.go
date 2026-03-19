package main

import (
	"context"
	"fmt"

	"github.com/DaiYuANg/arcgo/examples/kvx/shared"
	"github.com/DaiYuANg/arcgo/kvx"
	redisadapter "github.com/DaiYuANg/arcgo/kvx/adapter/redis"
)

func main() {
	ctx := context.Background()

	container, addr, err := shared.StartContainer(ctx, shared.RedisImage())
	must(err)
	defer func() { _ = container.Terminate(ctx) }()

	adapter, err := redisadapter.New(kvx.ClientOptions{Addrs: []string{addr}})
	must(err)
	defer func() { _ = adapter.Close() }()

	id1, err := adapter.XAdd(ctx, "demo:events", "*", map[string][]byte{
		"type": []byte("user.created"),
		"id":   []byte("u-1"),
	})
	must(err)

	id2, err := adapter.XAdd(ctx, "demo:events", "*", map[string][]byte{
		"type": []byte("user.updated"),
		"id":   []byte("u-1"),
	})
	must(err)

	entries, err := adapter.XRead(ctx, "demo:events", "0-0", 10)
	must(err)

	length, err := adapter.XLen(ctx, "demo:events")
	must(err)

	fmt.Printf("redis stream addr: %s\n", addr)
	fmt.Printf("entry ids: %s, %s\n", id1, id2)
	fmt.Printf("xlen: %d\n", length)
	fmt.Printf("read entries: %d\n", len(entries))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
