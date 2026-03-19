package main

import "fmt"

func main() {
	fmt.Println("Available kvx examples:")
	fmt.Println("  go run ./examples/kvx/hash_repository")
	fmt.Println("  go run ./examples/kvx/json_repository")
	fmt.Println("  go run ./examples/kvx/redis_adapter")
	fmt.Println("  go run ./examples/kvx/redis_hash")
	fmt.Println("  go run ./examples/kvx/redis_json")
	fmt.Println("  go run ./examples/kvx/redis_stream")
	fmt.Println("  go run ./examples/kvx/valkey_hash")
	fmt.Println("  go run ./examples/kvx/valkey_json")
	fmt.Println("  go run ./examples/kvx/valkey_stream")
}
