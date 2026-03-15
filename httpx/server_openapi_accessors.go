package httpx

import (
	"log/slog"

	"github.com/DaiYuANg/archgo/httpx/adapter"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-playground/validator/v10"
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
