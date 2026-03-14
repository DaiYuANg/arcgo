package httpapp

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/DaiYuANg/arcgo/observabilityx"
	"github.com/gofiber/fiber/v2"
)

func NewRequestObservabilityMiddleware(obs observabilityx.Observability) fiber.Handler {
	return func(c *fiber.Ctx) error {
		method := c.Method()
		path := c.Path()
		started := time.Now()

		ctx, span := obs.StartSpan(c.UserContext(), "rbac.http.request",
			observabilityx.String("method", method),
			observabilityx.String("path", path),
		)
		c.SetUserContext(ctx)

		err := c.Next()
		if err != nil {
			span.RecordError(err)
		}

		status := c.Response().StatusCode()
		route := routePattern(c)
		attrs := []observabilityx.Attribute{
			observabilityx.String("method", method),
			observabilityx.String("route", route),
			observabilityx.String("status", strconv.Itoa(status)),
		}

		obs.AddCounter(ctx, "rbac_http_requests_total", 1, attrs...)
		obs.RecordHistogram(ctx, "rbac_http_request_duration_ms", float64(time.Since(started).Milliseconds()), attrs...)
		span.End()
		return err
	}
}

func NewRequestLogMiddleware(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		started := time.Now()
		err := c.Next()

		logger.Info("http request",
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.String("route", routePattern(c)),
			slog.Int("status", c.Response().StatusCode()),
			slog.Duration("duration", time.Since(started)),
		)
		return err
	}
}

func routePattern(c *fiber.Ctx) string {
	if c == nil {
		return ""
	}
	route := c.Route()
	if route == nil || route.Path == "" {
		return c.Path()
	}
	return route.Path
}
