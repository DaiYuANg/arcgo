package operation

import (
	"context"
	"time"

	"github.com/DaiYuANg/archgo/observabilityx"
	"github.com/samber/lo"
)

func Begin(
	ctx context.Context,
	obs observabilityx.Observability,
	name string,
) (context.Context, observabilityx.Span, func()) {
	started := time.Now()
	ctx, span := obs.StartSpan(ctx, name)
	return ctx, span, func() {
		obs.RecordHistogram(ctx, "rbac_route_duration_ms", float64(time.Since(started).Milliseconds()),
			observabilityx.String("operation", name),
		)
	}
}

func CountRouteResult(
	ctx context.Context,
	obs observabilityx.Observability,
	route string,
	result string,
	attrs ...observabilityx.Attribute,
) {
	base := []observabilityx.Attribute{
		observabilityx.String("route", route),
		observabilityx.String("result", result),
	}
	obs.AddCounter(ctx, "rbac_route_total", 1, lo.Concat(base, attrs)...)
}
