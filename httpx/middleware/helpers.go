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
