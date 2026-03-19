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

	container, addr, err := shared.StartContainer(ctx, shared.ValkeyJSONImage())
	must(err)
	defer func() { _ = container.Terminate(ctx) }()

	adapter, err := valkeyadapter.New(kvx.ClientOptions{Addrs: []string{addr}})
	must(err)
	defer func() { _ = adapter.Close() }()

	must(adapter.JSONSet(ctx, "demo:user:u-1", "$", []byte(`{"id":"u-1","name":"Alice","roles":["admin"]}`), 0))

	document, err := adapter.JSONGet(ctx, "demo:user:u-1", "$")
	must(err)

	must(adapter.JSONSetField(ctx, "demo:user:u-1", "$.name", []byte(`"Alice Smith"`)))

	name, err := adapter.JSONGetField(ctx, "demo:user:u-1", "$.name")
	must(err)

	fmt.Printf("valkey json addr: %s\n", addr)
	fmt.Printf("document: %s\n", string(document))
	fmt.Printf("updated name: %s\n", string(name))
	fmt.Printf("image: %s\n", shared.ValkeyJSONImage())
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
