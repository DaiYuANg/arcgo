package db

import (
	"context"
	"log/slog"

	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/DaiYuANg/arcgo/dix"
	"github.com/DaiYuANg/arcgo/examples/dix/backend/config"
	"github.com/DaiYuANg/arcgo/examples/dix/backend/schema"
)

// Module wires the backend example database and schema services.
var Module = dix.NewModule("db",
	dix.Imports(config.Module),
	dix.Providers(
		dix.Provider2(func(cfg config.AppConfig, log *slog.Logger) *dbx.DB {
			database, err := OpenSQLite(cfg.DB.DSN, DefaultOpts(log)...)
			if err != nil {
				panic(err)
			}
			userSchema := schema.UserSchema{}
			users := dbx.MustSchema("users", userSchema)
			if _, err := database.AutoMigrate(context.Background(), users); err != nil {
				panic(err)
			}
			return database
		}),
		dix.Provider0(func() schema.UserSchema {
			s := schema.UserSchema{}
			return dbx.MustSchema("users", s)
		}),
	),
	dix.Hooks(
		dix.OnStop(func(_ context.Context, database *dbx.DB) error {
			return database.Close()
		}),
	),
)
