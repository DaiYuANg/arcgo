package main

import (
	"fmt"
	"os"

	"github.com/DaiYuANg/arcgo/examples/rbac_backend/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
