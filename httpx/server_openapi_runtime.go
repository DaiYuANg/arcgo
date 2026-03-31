package httpx

import (
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

func (s *Server) applyPendingHumaConfig() {
	api := s.HumaAPI()
	if api == nil {
		return
	}

	if s.accessLog {
		api.UseMiddleware(s.accessLogMiddleware())
	}

	middlewares := s.humaMiddlewares.Values()
	if len(middlewares) > 0 {
		api.UseMiddleware(middlewares...)
	}

	s.applyStoredOpenAPIPatches()
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

func (s *Server) accessLogMiddleware() func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		start := time.Now()
		next(ctx)

		status := ctx.Status()
		if status == 0 {
			status = http.StatusOK
		}

		url := ctx.URL()
		attrs := []any{
			"method", ctx.Method(),
			"path", url.Path,
			"status", status,
			"duration", time.Since(start),
		}

		if route, ok := s.matchRoute(ctx.Method(), url.Path); ok {
			attrs = append(attrs, "route", route.Path, "handler", route.HandlerName)
		}

		s.logger.Info("httpx request", attrs...)
	}
}

func ensureComponents(doc *huma.OpenAPI) *huma.Components {
	if doc.Components == nil {
		doc.Components = &huma.Components{}
	}
	return doc.Components
}
