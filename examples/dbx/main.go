// Package main lists the available dbx example entry points.
package main

import "fmt"

func main() {
	printLine("dbx examples:")
	printLine("  go run ./examples/dbx/basic")
	printLine("  go run ./examples/dbx/codec")
	printLine("  go run ./examples/dbx/mutation")
	printLine("  go run ./examples/dbx/query_advanced")
	printLine("  go run ./examples/dbx/relations")
	printLine("  go run ./examples/dbx/migration")
	printLine("  go run ./examples/dbx/pure_sql")
}

func printLine(text string) {
	if _, err := fmt.Println(text); err != nil {
		panic(err)
	}
}
