package middleware

import "net/http"

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func requestPath(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	return r.URL.Path
}

func requestEscapedPath(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	return r.URL.EscapedPath()
}

func requestURL(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	return r.URL.String()
}

func routePattern(r *http.Request, cfg config) string {
	if cfg.resolveRoutePattern != nil {
		if route := cfg.resolveRoutePattern(r); route != "" {
			return route
		}
	}
	return requestPath(r)
}
