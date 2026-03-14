package providers

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/observabilityx"
	promobs "github.com/DaiYuANg/arcgo/observabilityx/prometheus"
)

func NewPrometheusAdapter(logger *slog.Logger) *promobs.Adapter {
	return promobs.New(
		promobs.WithLogger(logger),
		promobs.WithNamespace("arcgo_rbac_example"),
	)
}

func NewObservability(logger *slog.Logger, prom *promobs.Adapter) observabilityx.Observability {
	return observabilityx.Multi(observabilityx.NopWithLogger(logger), prom)
}
