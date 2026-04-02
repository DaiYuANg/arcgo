package httpx

import (
	"context"
	"net/http"

	"github.com/DaiYuANg/arcgo/collectionx"
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
	Method      string                   `json:"method"`
	Path        string                   `json:"path"`
	HandlerName string                   `json:"handler_name"`
	Comment     string                   `json:"comment,omitempty"`
	Tags        collectionx.List[string] `json:"tags,omitempty"`
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

// SecurityOptions configures OpenAPI security schemes and default requirements.
type SecurityOptions struct {
	Schemes      map[string]*huma.SecurityScheme
	Requirements []map[string][]string
}
