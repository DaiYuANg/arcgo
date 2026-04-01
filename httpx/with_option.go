package httpx

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-playground/validator/v10"
	"github.com/samber/lo"
)

// WithAdapter configures related behavior.
func WithAdapter(a adapter.Host) ServerOption {
	return func(s *Server) {
		s.adapter = a
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
		if logger != nil {
			s.logger = logger
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
		s.openAPIPatches.Add(func(doc *huma.OpenAPI) {
			applyOpenAPIInfo(doc, title, version, description)
		})
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
		lo.ForEach(lo.Entries(opts.Schemes), func(entry lo.Entry[string, *huma.SecurityScheme], _ int) {
			if entry.Key == "" || entry.Value == nil {
				return
			}
			s.openAPIPatches.Add(func(doc *huma.OpenAPI) {
				components := ensureComponents(doc)
				if components.SecuritySchemes == nil {
					components.SecuritySchemes = map[string]*huma.SecurityScheme{}
				}
				components.SecuritySchemes[entry.Key] = cloneSecurityScheme(entry.Value)
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
		lo.ForEach(lo.Filter(headers, func(header *huma.Param, _ int) bool {
			return header != nil
		}), func(header *huma.Param, _ int) {
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

// WithValidator configures the request validator.
func WithValidator(v *validator.Validate) ServerOption {
	return func(s *Server) {
		s.validator = v
	}
}

func applyOpenAPIInfo(doc *huma.OpenAPI, title, version, description string) {
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
