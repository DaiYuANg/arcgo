package httpx

import (
	"log/slog"

	"github.com/DaiYuANg/archgo/httpx/adapter"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-playground/validator/v10"
)

// WithAdapter configures related behavior.
func WithAdapter(a adapter.Adapter) ServerOption {
	return func(s *Server) {
		s.adapter = a
		if s.logger != nil {
			withAdapterCapability(a, func(loggerAdapter adapter.LoggerConfigurer) {
				loggerAdapter.SetLogger(s.logger)
			})
		}
	}
}

// WithBasePath configures related behavior.
func WithBasePath(path string) ServerOption {
	return func(s *Server) {
		s.basePath = normalizeRoutePrefix(path)
	}
}

// WithLogger configures related behavior.
func WithLogger(logger *slog.Logger) ServerOption {
	return func(s *Server) {
		s.logger = logger
		if logger != nil {
			UseAdapter(s, func(loggerAdapter adapter.LoggerConfigurer) {
				loggerAdapter.SetLogger(logger)
			})
		}
	}
}

// WithPrintRoutes configures related behavior.
func WithPrintRoutes(enabled bool) ServerOption {
	return func(s *Server) {
		s.printRoutes = enabled
	}
}

// WithPanicRecover enables or disables panic recovery for typed handlers.
func WithPanicRecover(enabled bool) ServerOption {
	return func(s *Server) {
		s.panicRecover = enabled
	}
}

// WithAccessLog enables or disables request logging through the server logger.
func WithAccessLog(enabled bool) ServerOption {
	return func(s *Server) {
		s.accessLog = enabled
	}
}

// WithOpenAPIInfo sets top-level OpenAPI info fields for the server.
func WithOpenAPIInfo(title, version, description string) ServerOption {
	return func(s *Server) {
		s.humaOptions.Title = title
		s.humaOptions.Version = version
		s.humaOptions.Description = description

		patch := func(doc *huma.OpenAPI) {
			if doc == nil {
				return
			}
			if doc.Info == nil {
				doc.Info = &huma.Info{}
			}
			if title != "" {
				doc.Info.Title = title
			}
			if version != "" {
				doc.Info.Version = version
			}
			if description != "" {
				doc.Info.Description = description
			}
		}

		s.openAPIPatches.Add(patch)
	}
}

// WithHumaOptions applies low-level Huma-related configuration in one option.
func WithHumaOptions(opts HumaOptions) ServerOption {
	return func(s *Server) {
		if opts.Title != "" {
			s.humaOptions.Title = opts.Title
		}
		if opts.Version != "" {
			s.humaOptions.Version = opts.Version
		}
		if opts.Description != "" {
			s.humaOptions.Description = opts.Description
		}
		if opts.DocsPath != "" {
			s.humaOptions.DocsPath = opts.DocsPath
		}
		if opts.OpenAPIPath != "" {
			s.humaOptions.OpenAPIPath = opts.OpenAPIPath
		}
		if opts.SchemasPath != "" {
			s.humaOptions.SchemasPath = opts.SchemasPath
		}
		if opts.DocsRenderer != "" {
			s.humaOptions.DocsRenderer = opts.DocsRenderer
		}
		if opts.DisableDocsRoutes {
			s.humaOptions.DisableDocsRoutes = true
		}

		if opts.Title != "" || opts.Version != "" || opts.Description != "" {
			WithOpenAPIInfo(opts.Title, opts.Version, opts.Description)(s)
		}
	}
}

// WithDocs configures docs UI, OpenAPI, and schema routes.
func WithDocs(opts DocsOptions) ServerOption {
	return func(s *Server) {
		applyDocsOptionsToHumaOptions(&s.humaOptions, opts)
	}
}

// WithOpenAPIDocs enables or disables the built-in docs/OpenAPI/schema routes.
func WithOpenAPIDocs(enabled bool) ServerOption {
	return func(s *Server) {
		s.humaOptions.DisableDocsRoutes = !enabled
	}
}

// WithOpenAPIPatch appends a construction-time OpenAPI patch.
func WithOpenAPIPatch(fn func(*huma.OpenAPI)) ServerOption {
	return func(s *Server) {
		if fn == nil {
			return
		}
		s.openAPIPatches.Add(fn)
	}
}

// WithHumaMiddleware registers API-level Huma middleware for future requests.
func WithHumaMiddleware(middlewares ...func(huma.Context, func(huma.Context))) ServerOption {
	return func(s *Server) {
		if len(middlewares) == 0 {
			return
		}
		s.humaMiddlewares.Add(middlewares...)
	}
}

// WithSecurity registers security schemes and default top-level requirements.
func WithSecurity(opts SecurityOptions) ServerOption {
	return func(s *Server) {
		forEachValidSecurityScheme(opts.Schemes, func(name string, scheme *huma.SecurityScheme) {
			s.openAPIPatches.Add(func(doc *huma.OpenAPI) {
				components := ensureComponents(doc)
				if components.SecuritySchemes == nil {
					components.SecuritySchemes = map[string]*huma.SecurityScheme{}
				}
				components.SecuritySchemes[name] = cloneSecurityScheme(scheme)
			})
		})

		if len(opts.Requirements) > 0 {
			requirements := cloneSecurityRequirements(opts.Requirements)
			s.openAPIPatches.Add(func(doc *huma.OpenAPI) {
				doc.Security = cloneSecurityRequirements(requirements)
			})
		}
	}
}

// WithGlobalHeaders adds header parameters to future operations.
func WithGlobalHeaders(headers ...*huma.Param) ServerOption {
	return func(s *Server) {
		forEachNonNilHeader(headers, func(header *huma.Param) {
			cloned := cloneParam(header)
			cloned.In = "header"
			s.operationModifiers.Add(func(op *huma.Operation) {
				appendOperationParameter(op, cloned)
			})
		})
	}
}

// WithValidation enables related functionality.
func WithValidation() ServerOption {
	return func(s *Server) {
		if s.validator == nil {
			s.validator = validator.New(validator.WithRequiredStructEnabled())
		}
	}
}

// WithValidator closes related resources.
func WithValidator(v *validator.Validate) ServerOption {
	return func(s *Server) {
		s.validator = v
	}
}

func forEachValidSecurityScheme(
	schemes map[string]*huma.SecurityScheme,
	fn func(name string, scheme *huma.SecurityScheme),
) {
	if fn == nil {
		return
	}
	for name, scheme := range schemes {
		if name == "" || scheme == nil {
			continue
		}
		fn(name, scheme)
	}
}

func forEachNonNilHeader(headers []*huma.Param, fn func(header *huma.Param)) {
	if fn == nil {
		return
	}
	for _, header := range headers {
		if header == nil {
			continue
		}
		fn(header)
	}
}
