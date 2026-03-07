package httpx

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/samber/lo"
)

// GetRoutes returns related data.
func (s *Server) GetRoutes() []RouteInfo {
	return s.routesSnapshot()
}

// GetRoutesByMethod returns routes matching the given HTTP method.
func (s *Server) GetRoutesByMethod(method string) []RouteInfo {
	method = strings.ToUpper(method)
	return lo.Filter(s.routesSnapshot(), func(route RouteInfo, _ int) bool {
		return route.Method == method
	})
}

// GetRoutesByPath returns routes whose path starts with the given prefix.
func (s *Server) GetRoutesByPath(prefix string) []RouteInfo {
	if prefix == "" {
		return s.routesSnapshot()
	}
	return lo.Filter(s.routesSnapshot(), func(route RouteInfo, _ int) bool {
		return strings.HasPrefix(route.Path, prefix)
	})
}

// HasRoute reports whether a route has been registered.
func (s *Server) HasRoute(method, path string) bool {
	return s.routeKeys.Contains(routeKey(strings.ToUpper(method), path))
}

// RouteCount returns the number of unique registered routes.
func (s *Server) RouteCount() int {
	return s.routes.Len()
}

// printRoutesIfEnabled logs registered routes when route printing is enabled.
func (s *Server) printRoutesIfEnabled() {
	if !s.printRoutes {
		return
	}

	routes := s.routesSnapshot()
	s.logger.Info("Registered routes", slog.Int("count", len(routes)))
	lo.ForEach(routes, func(route RouteInfo, _ int) {
		s.logger.Info("  "+route.String(),
			slog.String("method", route.Method),
			slog.String("path", route.Path),
			slog.String("handler", route.HandlerName),
		)
	})
}

func (s *Server) addRoute(route RouteInfo) {
	key := routeKey(route.Method, route.Path)
	if s.routeKeys.Contains(key) {
		return
	}

	s.routeKeys.Add(key)
	s.routes.Add(route)
	s.printRoutesIfEnabled()
}

func (s *Server) routesSnapshot() []RouteInfo {
	return s.routes.Values()
}

func routeKey(method, path string) string {
	return method + " " + path
}

func (s *Server) matchRoute(method, path string) (RouteInfo, bool) {
	for _, route := range s.routesSnapshot() {
		if route.Method != method {
			continue
		}
		if route.Path == path || routePatternMatches(route.Path, path) {
			return route, true
		}
	}
	return RouteInfo{}, false
}

func routePatternMatches(pattern, path string) bool {
	pattern = strings.Trim(pattern, "/")
	path = strings.Trim(path, "/")

	if pattern == "" || path == "" {
		return pattern == path
	}

	patternSegments := strings.Split(pattern, "/")
	pathSegments := strings.Split(path, "/")
	if len(patternSegments) != len(pathSegments) {
		return false
	}

	for i, segment := range patternSegments {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			continue
		}
		if segment != pathSegments[i] {
			return false
		}
	}
	return true
}

type accessLogResponseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func newAccessLogResponseWriter(w http.ResponseWriter) *accessLogResponseWriter {
	return &accessLogResponseWriter{ResponseWriter: w}
}

func (w *accessLogResponseWriter) WriteHeader(status int) {
	if !w.wroteHeader {
		w.status = status
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *accessLogResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

func (w *accessLogResponseWriter) Status() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (w *accessLogResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *accessLogResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("httpx: response writer does not support hijacking")
	}
	return hijacker.Hijack()
}

func (w *accessLogResponseWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

func (w *accessLogResponseWriter) ReadFrom(r io.Reader) (int64, error) {
	readerFrom, ok := w.ResponseWriter.(io.ReaderFrom)
	if !ok {
		return io.Copy(w.ResponseWriter, r)
	}
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return readerFrom.ReadFrom(r)
}
