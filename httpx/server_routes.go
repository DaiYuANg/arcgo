package httpx

import (
	"fmt"
	"log/slog"
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
	if s == nil || method == "" {
		return nil
	}
	return s.routesByMethod.Get(method)
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
	if s == nil {
		return false
	}
	key := routeKey(strings.ToUpper(method), path)
	_, ok := s.routeExact.Get(key)
	return ok
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
	if s == nil {
		return
	}
	route.Method = strings.ToUpper(route.Method)
	key := routeKey(route.Method, route.Path)
	if _, loaded := s.routeExact.GetOrStore(key, route); loaded {
		return
	}

	s.routesByMethod.Put(route.Method, route)
	if isParameterizedRoute(route.Path) {
		s.parameterizedRouteMatcher(route.Method).Add(route.Path, route, s.routeSequence.Add(1))
	}

	s.routes.Add(route)
	s.printRoutesIfEnabled()
}

func (s *Server) validateRouteRegistration(method, path string) error {
	if s == nil {
		return fmt.Errorf("%w: server is nil", ErrRouteNotRegistered)
	}
	if s.IsFrozen() {
		return fmt.Errorf("%w: cannot register route %s %s", ErrServerFrozen, strings.ToUpper(method), path)
	}
	if s.HasRoute(method, path) {
		return fmt.Errorf("%w: %s %s", ErrRouteAlreadyExists, strings.ToUpper(method), path)
	}
	return nil
}

func (s *Server) routesSnapshot() []RouteInfo {
	return s.routes.Values()
}

func routeKey(method, path string) string {
	return method + " " + path
}

func (s *Server) matchRoute(method, path string) (RouteInfo, bool) {
	if s == nil {
		return RouteInfo{}, false
	}

	method = strings.ToUpper(method)
	key := routeKey(method, path)

	if route, ok := s.routeExact.Get(key); ok {
		return route, true
	}

	matcher, ok := s.routeMatchers.Get(method)
	if !ok || matcher == nil {
		return RouteInfo{}, false
	}
	return matcher.Match(path)
}

func isParameterizedRoute(path string) bool {
	return strings.Contains(path, "{") && strings.Contains(path, "}")
}

func (s *Server) parameterizedRouteMatcher(method string) *routeMatcher {
	if s == nil {
		return nil
	}

	matcher := newRouteMatcher()
	actual, _ := s.routeMatchers.GetOrStore(method, matcher)
	return actual
}
