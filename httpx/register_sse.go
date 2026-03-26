package httpx

import (
	"context"
	"fmt"

	"github.com/danielgtaylor/huma/v2"
	humasse "github.com/danielgtaylor/huma/v2/sse"
	"github.com/samber/lo"
)

// GetSSE registers a typed SSE GET handler on the server.
func GetSSE[I any](s ServerRuntime, path string, eventTypeMap map[string]any, handler SSEHandler[I], operationOptions ...OperationOption) error {
	return RouteSSEWithPolicies(s, MethodGet, path, eventTypeMap, handler, SSEPolicyOperation[I](operationOptions...))
}

// GroupGetSSE registers a typed SSE GET handler under a route group.
func GroupGetSSE[I any](g *Group, path string, eventTypeMap map[string]any, handler SSEHandler[I], operationOptions ...OperationOption) error {
	return GroupRouteSSEWithPolicies(g, MethodGet, path, eventTypeMap, handler, SSEPolicyOperation[I](operationOptions...))
}

// MustGetSSE registers an SSE GET route and panics if registration fails.
func MustGetSSE[I any](s ServerRuntime, path string, eventTypeMap map[string]any, handler SSEHandler[I], operationOptions ...OperationOption) {
	lo.Must0(GetSSE(s, path, eventTypeMap, handler, operationOptions...))
}

// MustGroupGetSSE registers a grouped SSE GET route and panics if registration fails.
func MustGroupGetSSE[I any](g *Group, path string, eventTypeMap map[string]any, handler SSEHandler[I], operationOptions ...OperationOption) {
	lo.Must0(GroupGetSSE(g, path, eventTypeMap, handler, operationOptions...))
}

func registerSSE[I any](
	s *Server,
	api huma.API,
	method, registerPath, fullPath string,
	eventTypeMap map[string]any,
	handler SSEHandler[I],
	operationOptions []OperationOption,
	policies []SSERoutePolicy[I],
) error {
	if s == nil {
		return fmt.Errorf("%w: server is nil", ErrRouteNotRegistered)
	}
	if api == nil {
		return ErrAdapterNotFound
	}
	if len(eventTypeMap) == 0 {
		return fmt.Errorf("%w: sse event map is empty", ErrRouteNotRegistered)
	}
	nilEventTypes := lo.FilterMap(lo.Entries(eventTypeMap), func(entry lo.Entry[string, any], _ int) (string, bool) {
		return entry.Key, entry.Value == nil
	})
	if len(nilEventTypes) > 0 {
		return fmt.Errorf("%w: sse event type is nil for event %q", ErrRouteNotRegistered, nilEventTypes[0])
	}
	if handler == nil {
		return fmt.Errorf("%w: sse handler is nil", ErrRouteNotRegistered)
	}
	wrappedHandler := applySSERoutePolicies(handler, policies)

	opID := defaultOperationID(method, fullPath)
	op := huma.Operation{
		OperationID: opID,
		Method:      method,
		Path:        registerPath,
	}

	lo.ForEach(operationOptions, func(opt OperationOption, _ int) {
		if opt != nil {
			opt(&op)
		}
	})
	applySSEPolicyOperations(&op, policies)

	lo.ForEach(s.operationModifiers.Values(), func(modifier func(*huma.Operation), _ int) {
		if modifier != nil {
			modifier(&op)
		}
	})

	s.openAPIMu.Lock()
	defer s.openAPIMu.Unlock()
	if err := s.validateRouteRegistration(method, fullPath); err != nil {
		return err
	}
	humasse.Register(api, op, eventTypeMap, func(ctx context.Context, input *I, send SSESender) {
		wrappedHandler(ctx, input, send)
	})

	s.addRoute(RouteInfo{
		Method:      method,
		Path:        fullPath,
		HandlerName: handlerName(handler),
	})

	return nil
}
