package httpx

import (
	"context"
	"net/http"

	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/danielgtaylor/huma/v2"
	humaconditional "github.com/danielgtaylor/huma/v2/conditional"
	humasse "github.com/danielgtaylor/huma/v2/sse"
)

// Docs renderer constants mirror Huma's built-in renderer options.
const (
	DocsRendererScalar            = huma.DocsRendererScalar
	DocsRendererStoplightElements = huma.DocsRendererStoplightElements
	DocsRendererSwaggerUI         = huma.DocsRendererSwaggerUI
)

// HTTP method aliases used by the route registration helpers.
const (
	MethodGet     = http.MethodGet
	MethodPost    = http.MethodPost
	MethodPut     = http.MethodPut
	MethodDelete  = http.MethodDelete
	MethodPatch   = http.MethodPatch
	MethodHead    = http.MethodHead
	MethodOptions = http.MethodOptions
)

// RouteInfo describes a registered route for diagnostics and tests.
type RouteInfo struct {
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	HandlerName string   `json:"handler_name"`
	Comment     string   `json:"comment,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// String returns related data.
func (r RouteInfo) String() string {
	return r.Method + " " + r.Path + " -> " + r.HandlerName
}

// TypedHandler is the typed handler signature used by `httpx` routes.
type TypedHandler[I, O any] func(ctx context.Context, input *I) (*O, error)

// ConditionalParams aliases Huma conditional request params.
type ConditionalParams = humaconditional.Params

// SSEMessage aliases Huma SSE message for streaming payloads.
type SSEMessage = humasse.Message

// SSESender aliases Huma SSE sender for streaming events.
type SSESender = humasse.Sender

// SSEHandler is the typed handler signature used by SSE routes.
type SSEHandler[I any] func(ctx context.Context, input *I, send SSESender)

// OperationOption mutates a Huma operation before registration.
type OperationOption func(*huma.Operation)

// HumaOptions configures Huma-backed OpenAPI and docs behavior.
type HumaOptions struct {
	// Title sets the OpenAPI info title.
	Title string
	// Version sets the OpenAPI info version.
	Version string
	// Description sets the OpenAPI info description.
	Description string
	// DocsPath sets the docs UI route.
	DocsPath string
	// OpenAPIPath sets the OpenAPI spec route prefix, without extension.
	OpenAPIPath string
	// SchemasPath sets the JSON schema route prefix.
	SchemasPath string
	// DocsRenderer selects the built-in docs renderer.
	DocsRenderer string
	// DisableDocsRoutes disables docs, OpenAPI, and schema routes.
	DisableDocsRoutes bool
}

// DefaultHumaOptions provides default behavior.
func DefaultHumaOptions() HumaOptions {
	return HumaOptions{
		Title:       "My API",
		Version:     "1.0.0",
		Description: "API Documentation",
		DocsPath:    "/docs",
		OpenAPIPath: "/openapi",
		SchemasPath: "/schemas",
	}
}

// ToAdapterHumaOptions converts package-level Huma options to adapter options.
func ToAdapterHumaOptions(opts HumaOptions) adapter.HumaOptions {
	return adapter.HumaOptions(opts)
}

// DocsOptions configures docs UI and OpenAPI/schema route exposure.
type DocsOptions struct {
	Enabled     bool
	DocsPath    string
	OpenAPIPath string
	SchemasPath string
	Renderer    string
}

// DefaultDocsOptions returns the default docs configuration used by httpx.
func DefaultDocsOptions() DocsOptions {
	return DocsOptions{
		Enabled:     true,
		DocsPath:    "/docs",
		OpenAPIPath: "/openapi",
		SchemasPath: "/schemas",
		Renderer:    DocsRendererStoplightElements,
	}
}

// SecurityOptions configures OpenAPI security schemes and default requirements.
type SecurityOptions struct {
	Schemes      map[string]*huma.SecurityScheme
	Requirements []map[string][]string
}
