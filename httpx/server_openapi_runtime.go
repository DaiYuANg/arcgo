package httpx

import (
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"
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
	lo.ForEach(lo.Filter(s.openAPIPatches.Values(), func(patch func(*huma.OpenAPI), _ int) bool {
		return patch != nil
	}), func(patch func(*huma.OpenAPI), _ int) {
		patch(openAPI)
	})
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

		route := mo.TupleToOption(s.matchRoute(ctx.Method(), url.Path))
		if route.IsPresent() {
			matched := route.MustGet()
			attrs = append(attrs, "route", matched.Path, "handler", matched.HandlerName)
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
