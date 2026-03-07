package httpx

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/collectionx/list"
	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-playground/validator/v10"
	"github.com/samber/lo"
)

// Adapter returns the underlying runtime adapter.
func (s *Server) Adapter() adapter.Adapter {
	return s.adapter
}

// Logger returns the server logger.
func (s *Server) Logger() *slog.Logger {
	return s.logger
}

// PanicRecoverEnabled reports whether typed handlers are wrapped with panic recovery.
func (s *Server) PanicRecoverEnabled() bool {
	return s != nil && s.panicRecover
}

// AccessLogEnabled reports whether requests are logged through the server logger.
func (s *Server) AccessLogEnabled() bool {
	return s != nil && s.accessLog
}

// Validator returns the configured request validator, if any.
func (s *Server) Validator() *validator.Validate {
	return s.validator
}

// HumaAPI exposes the underlying Huma API.
func (s *Server) HumaAPI() huma.API {
	if s == nil || s.adapter == nil {
		return nil
	}
	return s.adapter.HumaAPI()
}

// OpenAPI returns the underlying Huma OpenAPI document.
func (s *Server) OpenAPI() *huma.OpenAPI {
	api := s.HumaAPI()
	if api == nil {
		return nil
	}
	return api.OpenAPI()
}

// Docs returns the server's tracked docs configuration.
func (s *Server) Docs() DocsOptions {
	if s == nil {
		return DefaultDocsOptions()
	}

	docs := DefaultDocsOptions()
	docs.Enabled = !s.humaOptions.DisableDocsRoutes
	if s.humaOptions.DocsPath != "" {
		docs.DocsPath = s.humaOptions.DocsPath
	}
	if s.humaOptions.OpenAPIPath != "" {
		docs.OpenAPIPath = s.humaOptions.OpenAPIPath
	}
	if s.humaOptions.SchemasPath != "" {
		docs.SchemasPath = s.humaOptions.SchemasPath
	}
	if s.humaOptions.DocsRenderer != "" {
		docs.Renderer = s.humaOptions.DocsRenderer
	}
	return docs
}

// ConfigureDocs mutates the tracked docs config.
func (s *Server) ConfigureDocs(fn func(*DocsOptions)) {
	if s == nil || fn == nil {
		return
	}

	docs := s.Docs()
	fn(&docs)
	applyDocsOptionsToHumaOptions(&s.humaOptions, docs)
	if configurable, ok := s.adapter.(adapter.HumaOptionsConfigurer); ok {
		configurable.ConfigureHumaOptions(s.humaOptions)
	}
}

// ConfigureOpenAPI mutates the underlying Huma OpenAPI document.
func (s *Server) ConfigureOpenAPI(fn func(*huma.OpenAPI)) {
	if fn == nil {
		return
	}
	openAPI := s.OpenAPI()
	if openAPI == nil {
		return
	}
	fn(openAPI)
}

// PatchOpenAPI mutates the underlying Huma OpenAPI document.
func (s *Server) PatchOpenAPI(fn func(*huma.OpenAPI)) {
	s.ConfigureOpenAPI(fn)
}

// UseHumaMiddleware registers API-level Huma middleware.
func (s *Server) UseHumaMiddleware(middlewares ...func(huma.Context, func(huma.Context))) {
	if len(middlewares) == 0 {
		return
	}
	s.humaMiddlewares = append(s.humaMiddlewares, middlewares...)
	if api := s.HumaAPI(); api != nil {
		api.UseMiddleware(middlewares...)
	}
}

// UseOperationModifier registers a server-level operation modifier for future operations.
func (s *Server) UseOperationModifier(modifier func(*huma.Operation)) {
	if s == nil || modifier == nil {
		return
	}
	s.operationModifiers = append(s.operationModifiers, modifier)
}

// AddTag registers OpenAPI tag metadata.
func (s *Server) AddTag(tag *huma.Tag) {
	if s == nil || tag == nil || tag.Name == "" {
		return
	}
	s.ConfigureOpenAPI(func(doc *huma.OpenAPI) {
		if findTag(doc.Tags, tag.Name) >= 0 {
			doc.Tags[findTag(doc.Tags, tag.Name)] = cloneTag(tag)
			return
		}
		doc.Tags = append(doc.Tags, cloneTag(tag))
	})
}

// RegisterSecurityScheme registers an OpenAPI security scheme component.
func (s *Server) RegisterSecurityScheme(name string, scheme *huma.SecurityScheme) {
	if s == nil || name == "" || scheme == nil {
		return
	}
	s.ConfigureOpenAPI(func(doc *huma.OpenAPI) {
		components := ensureComponents(doc)
		if components.SecuritySchemes == nil {
			components.SecuritySchemes = map[string]*huma.SecurityScheme{}
		}
		components.SecuritySchemes[name] = cloneSecurityScheme(scheme)
	})
}

// SetDefaultSecurity configures top-level OpenAPI security requirements.
func (s *Server) SetDefaultSecurity(requirements ...map[string][]string) {
	if s == nil {
		return
	}
	s.ConfigureOpenAPI(func(doc *huma.OpenAPI) {
		doc.Security = cloneSecurityRequirements(requirements)
	})
}

// RegisterComponentParameter registers a reusable OpenAPI parameter component.
func (s *Server) RegisterComponentParameter(name string, param *huma.Param) {
	if s == nil || name == "" || param == nil {
		return
	}
	s.ConfigureOpenAPI(func(doc *huma.OpenAPI) {
		components := ensureComponents(doc)
		if components.Parameters == nil {
			components.Parameters = map[string]*huma.Param{}
		}
		components.Parameters[name] = cloneParam(param)
	})
}

// RegisterComponentHeader registers a reusable OpenAPI header component.
func (s *Server) RegisterComponentHeader(name string, header *huma.Param) {
	if s == nil || name == "" || header == nil {
		return
	}
	s.ConfigureOpenAPI(func(doc *huma.OpenAPI) {
		components := ensureComponents(doc)
		if components.Headers == nil {
			components.Headers = map[string]*huma.Header{}
		}
		components.Headers[name] = cloneParam(header)
	})
}

// RegisterGlobalParameter adds a parameter to all current and future operations.
func (s *Server) RegisterGlobalParameter(param *huma.Param) {
	if s == nil || param == nil || param.Name == "" || param.In == "" {
		return
	}

	cloned := cloneParam(param)
	s.UseOperationModifier(func(op *huma.Operation) {
		appendOperationParameter(op, cloned)
	})
	s.ConfigureOpenAPI(func(doc *huma.OpenAPI) {
		forEachOperation(doc, func(op *huma.Operation) {
			appendOperationParameter(op, cloned)
		})
	})
}

// RegisterGlobalHeader adds a request header parameter to all current and future operations.
func (s *Server) RegisterGlobalHeader(header *huma.Param) {
	if header == nil {
		return
	}
	cloned := cloneParam(header)
	cloned.In = "header"
	s.RegisterGlobalParameter(cloned)
}

func (s *Server) applyPendingHumaConfig() {
	if configurable, ok := s.adapter.(adapter.HumaOptionsConfigurer); ok {
		if !isZeroDocsOptions(s.humaOptions) {
			configurable.ConfigureHumaOptions(s.humaOptions)
		}
	}
	s.applyStoredOpenAPIPatches()
	if api := s.HumaAPI(); api != nil && len(s.humaMiddlewares) > 0 {
		api.UseMiddleware(s.humaMiddlewares...)
	}
}

func (s *Server) applyStoredOpenAPIPatches() {
	openAPI := s.OpenAPI()
	if openAPI == nil {
		return
	}
	for _, patch := range s.openAPIPatches {
		if patch != nil {
			patch(openAPI)
		}
	}
}

func applyDocsOptionsToHumaOptions(dst *adapter.HumaOptions, docs DocsOptions) {
	if dst == nil {
		return
	}
	dst.DisableDocsRoutes = !docs.Enabled
	if docs.DocsPath != "" {
		dst.DocsPath = docs.DocsPath
	}
	if docs.OpenAPIPath != "" {
		dst.OpenAPIPath = docs.OpenAPIPath
	}
	if docs.SchemasPath != "" {
		dst.SchemasPath = docs.SchemasPath
	}
	if docs.Renderer != "" {
		dst.DocsRenderer = docs.Renderer
	}
}

func ensureComponents(doc *huma.OpenAPI) *huma.Components {
	if doc.Components == nil {
		doc.Components = &huma.Components{}
	}
	return doc.Components
}

func forEachOperation(doc *huma.OpenAPI, fn func(*huma.Operation)) {
	if doc == nil || fn == nil {
		return
	}
	lo.ForEach(lo.Entries(doc.Paths), func(entry lo.Entry[string, *huma.PathItem], _ int) {
		if entry.Value == nil {
			return
		}
		list.NewList(
			entry.Value.Get, entry.Value.Put, entry.Value.Post, entry.Value.Delete,
			entry.Value.Options, entry.Value.Head, entry.Value.Patch, entry.Value.Trace,
		).Range(func(_ int, op *huma.Operation) bool {
			if op != nil {
				fn(op)
			}
			return true
		})
	})
}

func appendOperationParameter(op *huma.Operation, param *huma.Param) {
	if op == nil || param == nil {
		return
	}
	for _, existing := range op.Parameters {
		if existing != nil && existing.Name == param.Name && existing.In == param.In {
			return
		}
	}
	op.Parameters = append(op.Parameters, cloneParam(param))
}

func cloneParam(param *huma.Param) *huma.Param {
	if param == nil {
		return nil
	}
	cloned := *param
	if param.Schema != nil {
		schema := *param.Schema
		cloned.Schema = &schema
	}
	if param.Examples != nil {
		cloned.Examples = make(map[string]*huma.Example, len(param.Examples))
		for k, v := range param.Examples {
			cloned.Examples[k] = v
		}
	}
	if param.Extensions != nil {
		cloned.Extensions = make(map[string]any, len(param.Extensions))
		for k, v := range param.Extensions {
			cloned.Extensions[k] = v
		}
	}
	return &cloned
}

func cloneTag(tag *huma.Tag) *huma.Tag {
	if tag == nil {
		return nil
	}
	cloned := *tag
	if tag.Extensions != nil {
		cloned.Extensions = make(map[string]any, len(tag.Extensions))
		for k, v := range tag.Extensions {
			cloned.Extensions[k] = v
		}
	}
	return &cloned
}

func cloneExternalDocs(docs *huma.ExternalDocs) *huma.ExternalDocs {
	if docs == nil {
		return nil
	}
	cloned := *docs
	cloned.Extensions = cloneExtensions(docs.Extensions)
	return &cloned
}

func cloneSecurityScheme(scheme *huma.SecurityScheme) *huma.SecurityScheme {
	if scheme == nil {
		return nil
	}
	cloned := *scheme
	if scheme.Extensions != nil {
		cloned.Extensions = make(map[string]any, len(scheme.Extensions))
		for k, v := range scheme.Extensions {
			cloned.Extensions[k] = v
		}
	}
	return &cloned
}

func cloneExtensions(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(values))
	for k, v := range values {
		cloned[k] = v
	}
	return cloned
}

func cloneSecurityRequirements(requirements []map[string][]string) []map[string][]string {
	if len(requirements) == 0 {
		return nil
	}
	cloned := make([]map[string][]string, 0, len(requirements))
	for _, req := range requirements {
		if req == nil {
			cloned = append(cloned, nil)
			continue
		}
		item := make(map[string][]string, len(req))
		for k, scopes := range req {
			if scopes == nil {
				item[k] = []string{}
				continue
			}
			item[k] = append([]string(nil), scopes...)
		}
		cloned = append(cloned, item)
	}
	return cloned
}

func findTag(tags []*huma.Tag, name string) int {
	for i, tag := range tags {
		if tag != nil && tag.Name == name {
			return i
		}
	}
	return -1
}

func isZeroDocsOptions(opts adapter.HumaOptions) bool {
	return opts.DocsPath == "" &&
		opts.OpenAPIPath == "" &&
		opts.SchemasPath == "" &&
		opts.DocsRenderer == "" &&
		!opts.DisableDocsRoutes
}
