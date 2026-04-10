package middleware

import (
	"net/http"

	"github.com/DaiYuANg/arcgo/httpx"
)

type config struct {
	resolveRoutePattern func(*http.Request) string
}

// Option configures httpx middleware behavior.
type Option func(*config)

func applyOptions(opts []Option) config {
	cfg := config{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return cfg
}

// WithRoutePatternResolver configures route pattern resolution for metrics and traces.
func WithRoutePatternResolver(resolver func(*http.Request) string) Option {
	return func(cfg *config) {
		if cfg == nil {
			return
		}
		cfg.resolveRoutePattern = resolver
	}
}

// WithHTTPXRoutePattern resolves request paths through httpx route matching.
func WithHTTPXRoutePattern(server httpx.ServerRuntime) Option {
	return WithRoutePatternResolver(func(r *http.Request) string {
		if r == nil || server == nil {
			return ""
		}
		route, ok := server.MatchRoute(r.Method, requestPath(r))
		if !ok {
			return ""
		}
		return route.Path
	})
}
