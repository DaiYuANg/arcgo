// Package httpx_test exercises httpx through its public and test-only bridges.
//
//nolint:wrapcheck // Test-only generic forwarders preserve original errors for assertions.
package httpx_test

import (
	"context"
	"time"

	httpx "github.com/DaiYuANg/arcgo/httpx"
)

const (
	DocsRendererScalar = httpx.DocsRendererScalar
	MethodGet          = httpx.MethodGet
	MethodPost         = httpx.MethodPost
	MethodPut          = httpx.MethodPut
	MethodDelete       = httpx.MethodDelete
)

type (
	Server                      = httpx.Server
	ServerRuntime               = httpx.ServerRuntime
	Group                       = httpx.Group
	Registrar                   = httpx.Registrar
	BaseEndpoint                = httpx.BaseEndpoint
	Error                       = httpx.Error
	ConditionalParams           = httpx.ConditionalParams
	RouteInfo                   = httpx.RouteInfo
	EndpointSpec                = httpx.EndpointSpec
	OpenAPITags                 = httpx.OpenAPITags
	OpenAPITagDefinitions       = httpx.OpenAPITagDefinitions
	OpenAPIParameters           = httpx.OpenAPIParameters
	OpenAPIExtensions           = httpx.OpenAPIExtensions
	OpenAPISecuritySchemes      = httpx.OpenAPISecuritySchemes
	OpenAPISecurityRequirement  = httpx.OpenAPISecurityRequirement
	OpenAPISecurityRequirements = httpx.OpenAPISecurityRequirements
	AutoRoute                   = httpx.AutoRoute
	RoutePolicy[I, O any]       = httpx.RoutePolicy[I, O]
	TypedHandler[I, O any]      = httpx.TypedHandler[I, O]
	SSEHandler[I any]           = httpx.SSEHandler[I]
	SSESender                   = httpx.SSESender
	SecurityOptions             = httpx.SecurityOptions
	OperationOption             = httpx.OperationOption
	ServerOption                = httpx.ServerOption
	SSERoutePolicy[I any]       = httpx.SSERoutePolicy[I]
)

var (
	ErrAdapterNotFound    = httpx.ErrAdapterNotFound
	ErrInvalidHandlerName = httpx.ErrInvalidHandlerName
	ErrRouteAlreadyExists = httpx.ErrRouteAlreadyExists
	ErrServerFrozen       = httpx.ErrServerFrozen
	NewError              = httpx.NewError

	newServer                 = httpx.NewServerForTest
	newTestRequest            = httpx.NewRequestForTest
	serveRequest              = httpx.ServeRequestForTest
	matchRoute                = httpx.MatchRouteForTest
	normalizeRoutePrefix      = httpx.NormalizeRoutePrefixForTest
	joinRoutePath             = httpx.JoinRoutePathForTest
	freezeServer              = httpx.FreezeServerForTest
	BindGracefulShutdownHooks = httpx.BindGracefulShutdownHooks
	OperationConditionalRead  = httpx.OperationConditionalRead
	OperationConditionalWrite = httpx.OperationConditionalWrite
	Tags                      = httpx.Tags
	TagDefinitions            = httpx.TagDefinitions
	Parameters                = httpx.Parameters
	Extensions                = httpx.Extensions
	SecuritySchemes           = httpx.SecuritySchemes
	SecurityRequirement       = httpx.SecurityRequirement
	SecurityRequirementMap    = httpx.SecurityRequirementMap
	SecurityRequirements      = httpx.SecurityRequirements
	WithGlobalHeaders         = httpx.WithGlobalHeaders
	WithAdapter               = httpx.WithAdapter
	WithAccessLog             = httpx.WithAccessLog
	WithBasePath              = httpx.WithBasePath
	WithLogger                = httpx.WithLogger
	WithOpenAPIInfo           = httpx.WithOpenAPIInfo
	WithPanicRecover          = httpx.WithPanicRecover
	WithPrintRoutes           = httpx.WithPrintRoutes
	WithSecurity              = httpx.WithSecurity
	WithValidation            = httpx.WithValidation
	WithValidator             = httpx.WithValidator
)

func useHostCapability[T any](server ServerRuntime, use func(T)) bool {
	return httpx.UseHostCapabilityForTest[T](server, use)
}

func Get[I, O any](s ServerRuntime, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return httpx.Get(s, path, handler, operationOptions...)
}

func Post[I, O any](s ServerRuntime, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return httpx.Post(s, path, handler, operationOptions...)
}

func Put[I, O any](s ServerRuntime, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return httpx.Put(s, path, handler, operationOptions...)
}

func Delete[I, O any](s ServerRuntime, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return httpx.Delete(s, path, handler, operationOptions...)
}

func MustGet[I, O any](s ServerRuntime, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) {
	httpx.MustGet(s, path, handler, operationOptions...)
}

func GroupGet[I, O any](g *Group, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return httpx.GroupGet(g, path, handler, operationOptions...)
}

func MustGroupGet[I, O any](g *Group, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) {
	httpx.MustGroupGet(g, path, handler, operationOptions...)
}

func GroupRoute[I, O any](g *Group, method, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return httpx.GroupRoute(g, method, path, handler, operationOptions...)
}

func Auto[I, O any](handler TypedHandler[I, O], operationOptions ...OperationOption) AutoRoute {
	return httpx.Auto(handler, operationOptions...)
}

func RegisterAuto(registrar Registrar, routes ...AutoRoute) error {
	return httpx.RegisterAuto(registrar, routes...)
}

func MustAuto(registrar Registrar, routes ...AutoRoute) {
	httpx.MustAuto(registrar, routes...)
}

func GetSSE[I any](s ServerRuntime, path string, eventTypeMap map[string]any, handler SSEHandler[I], operationOptions ...OperationOption) error {
	return httpx.GetSSE(s, path, eventTypeMap, handler, operationOptions...)
}

func GroupGetSSE[I any](g *Group, path string, eventTypeMap map[string]any, handler SSEHandler[I], operationOptions ...OperationOption) error {
	return httpx.GroupGetSSE(g, path, eventTypeMap, handler, operationOptions...)
}

func RouteWithPolicies[I, O any](s ServerRuntime, method, path string, handler TypedHandler[I, O], policies ...RoutePolicy[I, O]) error {
	return httpx.RouteWithPolicies(s, method, path, handler, policies...)
}

func GroupRouteWithPolicies[I, O any](g *Group, method, path string, handler TypedHandler[I, O], policies ...RoutePolicy[I, O]) error {
	return httpx.GroupRouteWithPolicies(g, method, path, handler, policies...)
}

func RouteSSEWithPolicies[I any](s ServerRuntime, method, path string, eventTypeMap map[string]any, handler SSEHandler[I], policies ...SSERoutePolicy[I]) error {
	return httpx.RouteSSEWithPolicies(s, method, path, eventTypeMap, handler, policies...)
}

func GroupRouteSSEWithPolicies[I any](g *Group, method, path string, eventTypeMap map[string]any, handler SSEHandler[I], policies ...SSERoutePolicy[I]) error {
	return httpx.GroupRouteSSEWithPolicies(g, method, path, eventTypeMap, handler, policies...)
}

func PolicyConditionalRead[I, O any](stateGetter func(context.Context, *I) (string, time.Time, error)) RoutePolicy[I, O] {
	return httpx.PolicyConditionalRead[I, O](stateGetter)
}

func PolicyConditionalWrite[I, O any](stateGetter func(context.Context, *I) (string, time.Time, error)) RoutePolicy[I, O] {
	return httpx.PolicyConditionalWrite[I, O](stateGetter)
}

func PolicyHTMLResponse[I, O any]() RoutePolicy[I, O] {
	return httpx.PolicyHTMLResponse[I, O]()
}

func PolicyImageResponse[I, O any](contentTypes ...string) RoutePolicy[I, O] {
	return httpx.PolicyImageResponse[I, O](contentTypes...)
}

func PolicyOperation[I, O any](operationOptions ...OperationOption) RoutePolicy[I, O] {
	return httpx.PolicyOperation[I, O](operationOptions...)
}

func PolicyTimeout[I, O any](timeout time.Duration) RoutePolicy[I, O] {
	return httpx.PolicyTimeout[I, O](timeout)
}

func SSEPolicyOperation[I any](operationOptions ...OperationOption) SSERoutePolicy[I] {
	return httpx.SSEPolicyOperation[I](operationOptions...)
}

type pingOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

type echoInput struct {
	Body struct {
		Name string `json:"name"`
	}
}

type echoOutput struct {
	Body struct {
		Name string `json:"name"`
	}
}

type customBindInput struct {
	ID    int    `query:"user_id"`
	Token string `header:"X-Token"`
}

type customBindOutput struct {
	Body struct {
		ID    int    `json:"id"`
		Token string `json:"token"`
	}
}

type paramsInput struct {
	ID    int    `query:"id"`
	Flag  bool   `query:"flag"`
	Trace string `header:"X-Trace-ID"`
}

type paramsOutput struct {
	Body struct {
		ID    int    `json:"id"`
		Flag  bool   `json:"flag"`
		Trace string `json:"trace"`
	}
}

type validatedBodyInput struct {
	Body struct {
		Name string `json:"name" validate:"required,min=3"`
	}
}

type validatedBodyOutput struct {
	Body struct {
		Name string `json:"name"`
	}
}

type validatedQueryInput struct {
	ID int `query:"id" validate:"required,min=1"`
}

type customValidatedInput struct {
	Body struct {
		Name string `json:"name" validate:"arc"`
	}
}

type humaPingOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}
