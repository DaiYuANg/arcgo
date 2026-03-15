package httpx

import (
	"github.com/DaiYuANg/archgo/httpx/adapter"
	"github.com/danielgtaylor/huma/v2"
)

func (s *Server) applyPendingHumaConfig() {
	if !isZeroDocsOptions(s.humaOptions) {
		UseAdapter(s, func(configurable adapter.HumaOptionsConfigurer) {
			configurable.ConfigureHumaOptions(s.humaOptions)
		})
	}
	s.applyStoredOpenAPIPatches()
	if api := s.HumaAPI(); api != nil {
		middlewares := s.humaMiddlewares.Values()
		if len(middlewares) > 0 {
			api.UseMiddleware(middlewares...)
		}
	}
}

func (s *Server) applyStoredOpenAPIPatches() {
	openAPI := s.OpenAPI()
	if openAPI == nil {
		return
	}
	for _, patch := range s.openAPIPatches.Values() {
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

func isZeroDocsOptions(opts adapter.HumaOptions) bool {
	return opts.DocsPath == "" &&
		opts.OpenAPIPath == "" &&
		opts.SchemasPath == "" &&
		opts.DocsRenderer == "" &&
		!opts.DisableDocsRoutes
}
