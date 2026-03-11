package httpx

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-playground/validator/v10"
)

// ServerRuntime defines the public runtime contract exposed by httpx.
type ServerRuntime interface {
	http.Handler
	Handler() http.Handler
	ServeHTTP(http.ResponseWriter, *http.Request)
	ListenAndServe(addr string) error
	ListenAndServeContext(ctx context.Context, addr string) error

	Adapter() adapter.Adapter
	Logger() *slog.Logger
	PanicRecoverEnabled() bool
	AccessLogEnabled() bool
	Validator() *validator.Validate
	HumaAPI() huma.API
	OpenAPI() *huma.OpenAPI

	Docs() DocsOptions
	ConfigureDocs(fn func(*DocsOptions))
	ConfigureOpenAPI(fn func(*huma.OpenAPI))
	PatchOpenAPI(fn func(*huma.OpenAPI))
	UseOpenAPIPatch(fn func(*huma.OpenAPI))
	UseHumaMiddleware(...func(huma.Context, func(huma.Context)))
	UseOperationModifier(func(*huma.Operation))
	AddTag(*huma.Tag)
	RegisterSecurityScheme(name string, scheme *huma.SecurityScheme)
	SetDefaultSecurity(requirements ...map[string][]string)
	RegisterComponentParameter(name string, param *huma.Param)
	RegisterComponentHeader(name string, header *huma.Param)
	RegisterGlobalParameter(*huma.Param)
	RegisterGlobalHeader(*huma.Param)

	Group(prefix string) *Group
	GetRoutes() []RouteInfo
	GetRoutesByMethod(method string) []RouteInfo
	GetRoutesByPath(prefix string) []RouteInfo
	HasRoute(method, path string) bool
	RouteCount() int
	Register(endpoint Endpoint, hooks ...EndpointHooks)
	RegisterOnly(endpoints ...Endpoint)

	IsFrozen() bool
	asServer() *Server
}

// New creates a server exposed as the stable interface contract.
func New(opts ...ServerOption) ServerRuntime {
	return newServer(opts...)
}

var _ ServerRuntime = (*Server)(nil)

func (s *Server) asServer() *Server {
	return s
}

func unwrapServer(s ServerRuntime) *Server {
	if s == nil {
		return nil
	}
	return s.asServer()
}
