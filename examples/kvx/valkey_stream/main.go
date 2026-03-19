package main

import (
	"context"
	"fmt"

	"github.com/DaiYuANg/arcgo/examples/kvx/shared"
	"github.com/DaiYuANg/arcgo/kvx"
	valkeyadapter "github.com/DaiYuANg/arcgo/kvx/adapter/valkey"
)

func main() {
	ctx := context.Background()

	container, addr, err := shared.StartContainer(ctx, shared.ValkeyImage())
	must(err)
	defer func() { _ = container.Terminate(ctx) }()

	adapter, err := valkeyadapter.New(kvx.ClientOptions{Addrs: []string{addr}})
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

	fmt.Printf("valkey stream addr: %s\n", addr)
	fmt.Printf("entry ids: %s, %s\n", id1, id2)
	fmt.Printf("xlen: %d\n", length)
	fmt.Printf("read entries: %d\n", len(entries))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
