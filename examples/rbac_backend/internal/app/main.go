package app

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/examples/rbac_backend/pkg/appfx"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

// Run boots the rbac backend application and blocks until shutdown.
func Run() error {
	fxApp := fx.New(
		newAppModule(),
		fx.WithLogger(func(logger *slog.Logger) fxevent.Logger {
			return &fxevent.SlogLogger{Logger: logger}
		}),
	)

	return appfx.Run(fxApp)
}
