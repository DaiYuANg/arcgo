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

	must(adapter.HSet(ctx, "demo:user:u-1", map[string][]byte{
		"id":    []byte("u-1"),
		"name":  []byte("Alice"),
		"email": []byte("alice@example.com"),
	}))

	name, err := adapter.HGet(ctx, "demo:user:u-1", "name")
	must(err)

	fields, err := adapter.HGetAll(ctx, "demo:user:u-1")
	must(err)

	length, err := adapter.HLen(ctx, "demo:user:u-1")
	must(err)

	fmt.Printf("valkey hash addr: %s\n", addr)
	fmt.Printf("name: %s\n", string(name))
	fmt.Printf("fields: %d\n", len(fields))
	fmt.Printf("hlen: %d\n", length)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
