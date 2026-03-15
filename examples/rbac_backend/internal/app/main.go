package app

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/dix"
)

// Run boots the rbac backend application and blocks until shutdown.
func Run() error {
	app, err := dix.CreateApplicationContainer[*slog.Logger](
		newAppModule(),
	)
	if err != nil {
		return err
	}
	app.Run()
	return nil
}
