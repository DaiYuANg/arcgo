package httpx

import (
	"context"
	"log/slog"
	"reflect"

	"github.com/danielgtaylor/huma/v2"
	"github.com/samber/lo"
)

// Endpoint is an optional route-module interface for organizing related routes.
type Endpoint interface {
	RegisterRoutes(server ServerRuntime)
}

// GroupEndpoint registers routes against a scoped group prepared by httpx.
// It is optional and can be combined with EndpointSpecProvider for endpoint-level defaults.
type GroupEndpoint interface {
	RegisterGroupRoutes(group *Group)
}

// EndpointSpec describes optional endpoint-level group defaults applied before registration.
type EndpointSpec struct {
	Prefix        string
	Tags          []string
	Security      []map[string][]string
	Parameters    []*huma.Param
	SummaryPrefix string
	Description   string
	ExternalDocs  *huma.ExternalDocs
	Extensions    map[string]any
}

// EndpointSpecProvider exposes endpoint-level registration metadata.
type EndpointSpecProvider interface {
	EndpointSpec() EndpointSpec
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

	s.registerEndpointRoutes(endpoint)

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
		s.registerEndpointRoutes(e)
	})
}

func (s *Server) registerEndpointRoutes(endpoint Endpoint) {
	if s == nil || endpoint == nil {
		return
	}

	groupEndpoint, ok := endpoint.(GroupEndpoint)
	if !ok {
		s.warnIgnoredEndpointSpec(endpoint)
		endpoint.RegisterRoutes(s)
		return
	}

	spec, _ := endpointSpec(endpoint)
	group := s.Group(spec.Prefix)
	applyEndpointSpec(group, spec)
	groupEndpoint.RegisterGroupRoutes(group)
}

func (s *Server) warnIgnoredEndpointSpec(endpoint Endpoint) {
	spec, ok := endpointSpec(endpoint)
	if !ok || !spec.hasConfiguration() || s.logger == nil {
		return
	}

	s.logger.Warn("httpx endpoint spec ignored; endpoint does not implement GroupEndpoint",
		"endpoint_type", reflect.TypeOf(endpoint).String(),
	)
}

func endpointSpec(endpoint Endpoint) (EndpointSpec, bool) {
	provider, ok := endpoint.(EndpointSpecProvider)
	if !ok || provider == nil {
		return EndpointSpec{}, false
	}
	return provider.EndpointSpec(), true
}

func applyEndpointSpec(group *Group, spec EndpointSpec) {
	if group == nil {
		return
	}
	if len(spec.Tags) > 0 {
		group.DefaultTags(spec.Tags...)
	}
	if len(spec.Security) > 0 {
		group.DefaultSecurity(spec.Security...)
	}
	if len(spec.Parameters) > 0 {
		group.DefaultParameters(spec.Parameters...)
	}
	if spec.SummaryPrefix != "" {
		group.DefaultSummaryPrefix(spec.SummaryPrefix)
	}
	if spec.Description != "" {
		group.DefaultDescription(spec.Description)
	}
	if spec.ExternalDocs != nil {
		group.DefaultExternalDocs(spec.ExternalDocs)
	}
	if len(spec.Extensions) > 0 {
		group.DefaultExtensions(spec.Extensions)
	}
}

func (s EndpointSpec) hasConfiguration() bool {
	return s.Prefix != "" ||
		len(s.Tags) > 0 ||
		len(s.Security) > 0 ||
		len(s.Parameters) > 0 ||
		s.SummaryPrefix != "" ||
		s.Description != "" ||
		s.ExternalDocs != nil ||
		len(s.Extensions) > 0
}
