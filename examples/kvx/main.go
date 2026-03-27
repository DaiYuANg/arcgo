// Package main lists the available kvx example entrypoints.
package main

import "fmt"

func main() {
	mustPrintln("Available kvx examples:")
	mustPrintln("  go run ./examples/kvx/hash_repository")
	mustPrintln("  go run ./examples/kvx/json_repository")
	mustPrintln("  go run ./examples/kvx/redis_adapter")
	mustPrintln("  go run ./examples/kvx/redis_hash")
	mustPrintln("  go run ./examples/kvx/redis_json")
	mustPrintln("  go run ./examples/kvx/redis_stream")
	mustPrintln("  go run ./examples/kvx/valkey_hash")
	mustPrintln("  go run ./examples/kvx/valkey_json")
	mustPrintln("  go run ./examples/kvx/valkey_stream")
}

func mustPrintln(args ...any) {
	if _, err := fmt.Println(args...); err != nil {
		panic(err)
	}
}
