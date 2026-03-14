package eventpublish

import (
	"context"
	"log/slog"

	"github.com/DaiYuANg/arcgo/eventx"
	"github.com/DaiYuANg/arcgo/logx"
)

func Async(ctx context.Context, bus eventx.BusRuntime, event eventx.Event, logger *slog.Logger) {
	if event == nil {
		return
	}
	if err := bus.PublishAsync(ctx, event); err != nil {
		logx.WithError(logx.WithFields(logger, map[string]any{"event": event.Name()}), err).
			Warn("publish async event failed")
	}
}
