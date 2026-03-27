// Package repo provides persistence services for the backend example.
package repo

import (
	"github.com/DaiYuANg/arcgo/dix"
	"github.com/DaiYuANg/arcgo/examples/dix/backend/db"
)

// Module wires the backend example repositories.
var Module = dix.NewModule("repo",
	dix.WithModuleImports(db.Module),
	dix.WithModuleProviders(
		dix.Provider2(NewUserRepository),
	),
)
