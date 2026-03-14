package app

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/pkg/fxx"
)

// Run boots the rbac backend application and blocks until shutdown.
func Run() error {
	app, err := fxx.CreateApplicationContainer[*slog.Logger](
		newAppModule(),
	)
	if err != nil {
		return nil
	}
	app.Run()
	return nil
}
