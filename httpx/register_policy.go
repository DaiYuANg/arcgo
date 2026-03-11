package httpx

import (
	"fmt"

	"github.com/samber/lo"
)

// RouteWithPolicies registers a typed handler with runtime/OpenAPI policies.
func RouteWithPolicies[I, O any](s ServerRuntime, method, path string, handler TypedHandler[I, O], policies ...RoutePolicy[I, O]) error {
	server := unwrapServer(s)
	if server == nil {
		return fmt.Errorf("%w: server is nil", ErrRouteNotRegistered)
	}
	fullPath := joinRoutePath(server.basePath, path)
	return registerTyped(server, server.HumaAPI(), method, fullPath, fullPath, handler, nil, policies)
}

// GroupRouteWithPolicies registers a grouped typed handler with policies.
func GroupRouteWithPolicies[I, O any](g *Group, method, path string, handler TypedHandler[I, O], policies ...RoutePolicy[I, O]) error {
	if g == nil || g.server == nil {
		return fmt.Errorf("%w: route group is nil", ErrRouteNotRegistered)
	}
	fullPath := joinRoutePath(g.server.basePath, joinRoutePath(g.prefix, path))

	target := g.server.HumaAPI()
	registerPath := fullPath
	if g.humaGroup != nil {
		target = g.humaGroup
		registerPath = path
	}

	return registerTyped(g.server, target, method, registerPath, fullPath, handler, nil, policies)
}

// MustRouteWithPolicies registers a route with policies and panics on failure.
func MustRouteWithPolicies[I, O any](s ServerRuntime, method, path string, handler TypedHandler[I, O], policies ...RoutePolicy[I, O]) {
	lo.Must0(RouteWithPolicies(s, method, path, handler, policies...))
}

// MustGroupRouteWithPolicies registers a grouped route with policies and panics on failure.
func MustGroupRouteWithPolicies[I, O any](g *Group, method, path string, handler TypedHandler[I, O], policies ...RoutePolicy[I, O]) {
	lo.Must0(GroupRouteWithPolicies(g, method, path, handler, policies...))
}
