// Package adapter provides package-level APIs.
package adapter

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/samber/lo"
)

// HandlerFunc is the adapter-level handler signature used by native adapters.
type HandlerFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request) error

// MiddlewareFunc is the adapter-level middleware signature.
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

type routeParamsCtxKey struct{}

// Adapter is the runtime abstraction implemented by framework adapters.
type Adapter interface {
	// Name returns the adapter name, such as `std`, `gin`, or `echo`.
	Name() string

	// Handle registers a native handler on the underlying runtime.
	Handle(method, path string, handler HandlerFunc)

	// Group returns a child adapter scoped to the given prefix.
	Group(prefix string) Adapter

	// ServeHTTP serves requests for adapters that support `net/http`.
	ServeHTTP(w http.ResponseWriter, r *http.Request)

	// HumaAPI exposes the underlying Huma API.
	HumaAPI() huma.API
}

// RouterAdapter exposes the underlying router/engine/app with strong typing.
type RouterAdapter[R any] interface {
	Adapter
	Router() R
}

// ListenableAdapter is implemented by adapters that can listen directly.
type ListenableAdapter interface {
	Listen(addr string) error
}

// ContextListenableAdapter is implemented by adapters that support context-aware shutdown.
type ContextListenableAdapter interface {
	ListenContext(ctx context.Context, addr string) error
}

// LoggerConfigurer is implemented by adapters that accept an injected slog logger.
type LoggerConfigurer interface {
	SetLogger(*slog.Logger)
}

// HumaOptions configures Huma-backed metadata and docs exposure for an adapter.
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
	// SchemasPath sets the schema route prefix.
	SchemasPath string
	// DocsRenderer selects the built-in docs renderer.
	DocsRenderer string
	// DisableDocsRoutes disables docs, OpenAPI, and schema routes.
	DisableDocsRoutes bool
	// Transformers modifies response bodies before serialization.
	Transformers []huma.Transformer
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

// MergeHumaOptions merges multiple HumaOptions into one.
// Later options override earlier options for non-empty fields.
func MergeHumaOptions(opts ...HumaOptions) HumaOptions {
	result := DefaultHumaOptions()
	lo.ForEach(opts, func(opt HumaOptions, _ int) {
		result.Title = lo.Ternary(opt.Title != "", opt.Title, result.Title)
		result.Version = lo.Ternary(opt.Version != "", opt.Version, result.Version)
		result.Description = lo.Ternary(opt.Description != "", opt.Description, result.Description)
		result.DocsPath = lo.Ternary(opt.DocsPath != "", opt.DocsPath, result.DocsPath)
		result.OpenAPIPath = lo.Ternary(opt.OpenAPIPath != "", opt.OpenAPIPath, result.OpenAPIPath)
		result.SchemasPath = lo.Ternary(opt.SchemasPath != "", opt.SchemasPath, result.SchemasPath)
		result.DocsRenderer = lo.Ternary(opt.DocsRenderer != "", opt.DocsRenderer, result.DocsRenderer)
		result.DisableDocsRoutes = lo.Ternary(opt.DisableDocsRoutes, true, result.DisableDocsRoutes)
		if len(opt.Transformers) > 0 {
			result.Transformers = append(result.Transformers, opt.Transformers...)
		}
	})
	return result
}

// ApplyHumaConfig copies adapter Huma options into a Huma config.
func ApplyHumaConfig(cfg *huma.Config, opts HumaOptions) {
	if cfg == nil {
		return
	}

	cfg.Info.Description = opts.Description

	if opts.DisableDocsRoutes {
		cfg.DocsPath = ""
		cfg.OpenAPIPath = ""
		cfg.SchemasPath = ""
		return
	}

	cfg.DocsPath = normalizeDocsPath(opts.DocsPath)
	cfg.OpenAPIPath = normalizeOpenAPIPath(opts.OpenAPIPath)
	cfg.SchemasPath = normalizeSchemasPath(opts.SchemasPath)
	if opts.DocsRenderer != "" {
		cfg.DocsRenderer = opts.DocsRenderer
	}
	if len(opts.Transformers) > 0 {
		cfg.Transformers = append(cfg.Transformers, opts.Transformers...)
	}
}

func normalizeDocsPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/docs"
	}
	return ensureLeadingSlash(trimmed)
}

func normalizeOpenAPIPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/openapi"
	}

	trimmed = strings.TrimSuffix(trimmed, ".json")
	trimmed = strings.TrimSuffix(trimmed, ".yaml")
	trimmed = strings.TrimSuffix(trimmed, "/")
	if trimmed == "" {
		return "/openapi"
	}
	return ensureLeadingSlash(trimmed)
}

func normalizeSchemasPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/schemas"
	}

	trimmed = strings.TrimSuffix(trimmed, "/")
	if trimmed == "" {
		return "/schemas"
	}
	return ensureLeadingSlash(trimmed)
}

func ensureLeadingSlash(path string) string {
	if strings.HasPrefix(path, "/") {
		return path
	}
	return "/" + path
}

// WithRouteParams stores adapter-extracted path parameters on the context.
func WithRouteParams(ctx context.Context, params map[string]string) context.Context {
	if len(params) == 0 {
		return ctx
	}
	return context.WithValue(ctx, routeParamsCtxKey{}, params)
}

// RouteParam retrieves a path parameter previously stored on the context.
func RouteParam(ctx context.Context, name string) string {
	if ctx == nil || name == "" {
		return ""
	}

	params, ok := ctx.Value(routeParamsCtxKey{}).(map[string]string)
	if !ok {
		return ""
	}
	return params[name]
}
