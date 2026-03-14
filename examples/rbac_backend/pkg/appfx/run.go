package appfx

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/fx"
)

// Run starts an fx app, waits for shutdown signal, and stops gracefully.
func Run(app *fx.App) error {
	if app == nil {
		return fmt.Errorf("nil fx app")
	}

	startCtx, startCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer startCancel()
	if err := app.Start(startCtx); err != nil {
		return fmt.Errorf("start app failed: %w", err)
	}

	<-app.Done()

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer stopCancel()
	if err := app.Stop(stopCtx); err != nil {
		return fmt.Errorf("stop app failed: %w", err)
	}
	return nil
}
