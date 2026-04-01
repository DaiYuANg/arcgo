package httpx

import (
	"context"
	"log/slog"
	"reflect"

	"github.com/samber/lo"
)

// Endpoint is an optional route-module interface for organizing related routes.
type Endpoint interface {
	RegisterRoutes(server ServerRuntime)
}

// BaseEndpoint provides a no-op `RegisterRoutes` implementation for embedding.
type BaseEndpoint struct{}

// RegisterRoutes is a no-op default implementation.
func (e *BaseEndpoint) RegisterRoutes(_ ServerRuntime) {}

// EndpointHookFunc runs before or after endpoint registration.
type EndpointHookFunc func(server ServerRuntime, endpoint Endpoint)

// EndpointHooks wraps optional before/after endpoint registration hooks.
type EndpointHooks struct {
	Before EndpointHookFunc
	After  EndpointHookFunc
}

// Register registers one endpoint and runs any provided hooks around it.
func (s *Server) Register(endpoint Endpoint, hooks ...EndpointHooks) {
	if endpoint == nil {
		return
	}
	if s != nil && s.logger != nil && s.logger.Enabled(context.Background(), slog.LevelDebug) {
		s.logger.Debug("httpx endpoint registration starting",
			"endpoint_type", reflect.TypeOf(endpoint).String(),
			"hooks", len(hooks),
		)
	}

	runEndpointHooks(s, endpoint, hooks, func(h EndpointHooks) EndpointHookFunc { return h.Before })

	endpoint.RegisterRoutes(s)

	runEndpointHooks(s, endpoint, hooks, func(h EndpointHooks) EndpointHookFunc { return h.After })
	if s != nil && s.logger != nil && s.logger.Enabled(context.Background(), slog.LevelDebug) {
		s.logger.Debug("httpx endpoint registration completed",
			"endpoint_type", reflect.TypeOf(endpoint).String(),
			"routes", s.RouteCount(),
		)
	}
}

// RegisterOnly registers endpoints without hook processing.
func (s *Server) RegisterOnly(endpoints ...Endpoint) {
	lo.ForEach(endpoints, func(e Endpoint, _ int) {
		if e == nil {
			if s.logger != nil {
				s.logger.Warn("skipping nil endpoint")
			}
			return
		}
		e.RegisterRoutes(s)
	})
}
