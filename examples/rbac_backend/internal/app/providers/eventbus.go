package providers

import (
	"context"
	"log/slog"

	"github.com/DaiYuANg/archgo/eventx"
	"github.com/DaiYuANg/archgo/examples/rbac_backend/internal/config"
	"github.com/DaiYuANg/archgo/logx"
	"github.com/DaiYuANg/archgo/observabilityx"
	"go.uber.org/fx"
)

func NewEventBus(lc fx.Lifecycle, cfg config.AppConfig, obs observabilityx.Observability, logger *slog.Logger) eventx.BusRuntime {
	bus := eventx.New(
		eventx.WithObservability(obs),
		eventx.WithAntsPool(cfg.Event.Workers),
		eventx.WithParallelDispatch(cfg.Event.Parallel),
		eventx.WithAsyncErrorHandler(func(ctx context.Context, event eventx.Event, err error) {
			if err == nil || event == nil {
				return
			}
			logx.WithError(logx.WithFields(logger, map[string]any{
				"event": event.Name(),
			}), err).Error("async event dispatch failed")
		}),
	)

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return bus.Close()
		},
	})
	return bus
}
