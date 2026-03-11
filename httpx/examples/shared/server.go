package shared

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/httpx"
	"github.com/DaiYuANg/arcgo/httpx/adapter"
)

func NewRuntime(a adapter.Adapter, logger *slog.Logger) httpx.ServerRuntime {
	return httpx.New(
		httpx.WithAdapter(a),
		httpx.WithLogger(logger),
		httpx.WithPrintRoutes(true),
		httpx.WithValidation(),
	)
}
