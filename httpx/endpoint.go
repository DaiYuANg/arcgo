package httpx

import (
	"context"
	"log/slog"
	"reflect"

	"github.com/danielgtaylor/huma/v2"
	"github.com/samber/lo"
)

// Registrar is the narrow registration scope exposed to endpoint modules.
// Implementations provide a current scoped group and support nested groups.
type Registrar interface {
	Scope() *Group
	Group(prefix string) *Group
}

// Endpoint is the preferred route-module interface for organizing related routes.
// Endpoints receive a narrow registrar instead of the full server runtime.
type Endpoint interface {
	Register(registrar Registrar)
}

// LegacyEndpoint is the original endpoint contract that exposes the full server runtime.
// It remains supported for compatibility, but new endpoint modules should prefer Endpoint.
type LegacyEndpoint interface {
	RegisterRoutes(server ServerRuntime)
}

// GroupEndpoint registers routes against a scoped group prepared by httpx.
// It is optional and can be combined with EndpointSpecProvider for endpoint-level defaults.
//
// Deprecated: prefer Endpoint and Register(registrar) for new code.
type GroupEndpoint interface {
	RegisterGroupRoutes(group *Group)
}

// EndpointSpec describes optional endpoint-level group defaults applied before registration.
type EndpointSpec struct {
	Prefix        string
	Tags          OpenAPITags
	Security      OpenAPISecurityRequirements
	Parameters    OpenAPIParameters
	SummaryPrefix string
	Description   string
	ExternalDocs  *huma.ExternalDocs
	Extensions    OpenAPIExtensions
}

// EndpointSpecProvider exposes endpoint-level registration metadata.
type EndpointSpecProvider interface {
	EndpointSpec() EndpointSpec
}

// BaseEndpoint provides a no-op legacy RegisterRoutes implementation for embedding.
//
// Deprecated: new endpoint modules should implement Endpoint directly.
type BaseEndpoint struct{}

// RegisterRoutes is a no-op default implementation.
func (e *BaseEndpoint) RegisterRoutes(_ ServerRuntime) {}

// EndpointHookFunc runs before or after endpoint registration.
type EndpointHookFunc func(server ServerRuntime, endpoint any)

// EndpointHooks wraps optional before/after endpoint registration hooks.
type EndpointHooks struct {
	Before EndpointHookFunc
	After  EndpointHookFunc
}

// Register registers one endpoint and runs any provided hooks around it.
func (s *Server) Register(endpoint any, hooks ...EndpointHooks) {
	if isNilEndpoint(endpoint) {
		return
	}
	if s != nil && s.logger != nil && s.logger.Enabled(context.Background(), slog.LevelDebug) {
		s.logger.Debug("httpx endpoint registration starting",
			"endpoint_type", endpointTypeName(endpoint),
			"hooks", len(hooks),
		)
	}

	runEndpointHooks(s, endpoint, hooks, func(h EndpointHooks) EndpointHookFunc { return h.Before })

	s.registerEndpointRoutes(endpoint)

	runEndpointHooks(s, endpoint, hooks, func(h EndpointHooks) EndpointHookFunc { return h.After })
	if s != nil && s.logger != nil && s.logger.Enabled(context.Background(), slog.LevelDebug) {
		s.logger.Debug("httpx endpoint registration completed",
			"endpoint_type", endpointTypeName(endpoint),
			"routes", s.RouteCount(),
		)
	}
}

// RegisterOnly registers endpoints without hook processing.
func (s *Server) RegisterOnly(endpoints ...any) {
	lo.ForEach(endpoints, func(e any, _ int) {
		if isNilEndpoint(e) {
			if s.logger != nil {
				s.logger.Warn("skipping nil endpoint")
			}
			return
		}
		s.registerEndpointRoutes(e)
	})
}

func (s *Server) registerEndpointRoutes(endpoint any) {
	if s == nil || endpoint == nil {
		return
	}

	if registrarEndpoint, ok := endpoint.(Endpoint); ok {
		spec, _ := endpointSpec(endpoint)
		group := s.Group(spec.Prefix)
		applyEndpointSpec(group, spec)
		registrarEndpoint.Register(group)
		return
	}

	groupEndpoint, ok := endpoint.(GroupEndpoint)
	if !ok {
		s.warnIgnoredEndpointSpec(endpoint)
		legacyEndpoint, legacyOK := endpoint.(LegacyEndpoint)
		if legacyOK {
			legacyEndpoint.RegisterRoutes(s)
			return
		}
		if s.logger != nil {
			s.logger.Warn("skipping unsupported endpoint",
				"endpoint_type", endpointTypeName(endpoint),
			)
		}
		return
	}

	spec, _ := endpointSpec(endpoint)
	group := s.Group(spec.Prefix)
	applyEndpointSpec(group, spec)
	groupEndpoint.RegisterGroupRoutes(group)
}

func (s *Server) warnIgnoredEndpointSpec(endpoint any) {
	spec, ok := endpointSpec(endpoint)
	if !ok || !spec.hasConfiguration() || s.logger == nil {
		return
	}

	s.logger.Warn("httpx endpoint spec ignored; endpoint does not implement Endpoint or GroupEndpoint",
		"endpoint_type", endpointTypeName(endpoint),
	)
}

func endpointSpec(endpoint any) (EndpointSpec, bool) {
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
	if !spec.Tags.IsEmpty() {
		group.DefaultTags(spec.Tags)
	}
	if !spec.Security.IsEmpty() {
		group.DefaultSecurity(spec.Security)
	}
	if !spec.Parameters.IsEmpty() {
		group.DefaultParameters(spec.Parameters)
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
	if !spec.Extensions.IsEmpty() {
		group.DefaultExtensions(spec.Extensions)
	}
}

func (s EndpointSpec) hasConfiguration() bool {
	return s.Prefix != "" ||
		!s.Tags.IsEmpty() ||
		!s.Security.IsEmpty() ||
		!s.Parameters.IsEmpty() ||
		s.SummaryPrefix != "" ||
		s.Description != "" ||
		s.ExternalDocs != nil ||
		!s.Extensions.IsEmpty()
}

func isNilEndpoint(endpoint any) bool {
	if endpoint == nil {
		return true
	}
	value := reflect.ValueOf(endpoint)
	//nolint:exhaustive // Only nil-able reflect kinds can be checked with IsNil.
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func endpointTypeName(endpoint any) string {
	if endpoint == nil {
		return "<nil>"
	}
	t := reflect.TypeOf(endpoint)
	if t == nil {
		return "<nil>"
	}
	return t.String()
}
