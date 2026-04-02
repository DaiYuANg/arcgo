package httpx

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/DaiYuANg/arcgo/pkg/option"
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
	if err := validateSSERouteRegistration(eventTypeMap, handler); err != nil {
		return err
	}
	wrappedHandler := applySSERoutePolicies(handler, policies)
	op := newSSEOperation(method, registerPath, fullPath, operationOptions, policies)
	applyOperationModifiers(&op, s.operationModifiers)
	if s.logger != nil && s.logger.Enabled(context.Background(), slog.LevelDebug) {
		s.logger.Debug("httpx sse route registration starting",
			"method", method,
			"path", fullPath,
			"register_path", registerPath,
			"operation_id", op.OperationID,
			"event_types", len(eventTypeMap),
			"policies", len(policies),
		)
	}

	s.openAPIMu.Lock()
	defer s.openAPIMu.Unlock()
	if err := s.validateRouteRegistration(method, fullPath); err != nil {
		if s.logger != nil {
			s.logger.Error("httpx sse route registration failed",
				"method", method,
				"path", fullPath,
				"error", err,
			)
		}
		return err
	}
	humasse.Register(api, op, eventTypeMap, func(ctx context.Context, input *I, send SSESender) {
		wrappedHandler(ctx, input, send)
	})

	s.addRoute(RouteInfo{
		Method:      method,
		Path:        fullPath,
		HandlerName: handlerName(handler),
		Tags:        routeTags(op.Tags),
	})
	if s.logger != nil && s.logger.Enabled(context.Background(), slog.LevelDebug) {
		s.logger.Debug("httpx sse route registration completed",
			"method", method,
			"path", fullPath,
			"operation_id", op.OperationID,
		)
	}

	return nil
}

func validateSSERouteRegistration[I any](eventTypeMap map[string]any, handler SSEHandler[I]) error {
	if len(eventTypeMap) == 0 {
		return fmt.Errorf("%w: sse event map is empty", ErrRouteNotRegistered)
	}
	if invalidEvent, ok := lo.Find(lo.Entries(eventTypeMap), func(entry lo.Entry[string, any]) bool {
		return entry.Value == nil
	}); ok {
		return fmt.Errorf("%w: sse event type is nil for event %q", ErrRouteNotRegistered, invalidEvent.Key)
	}
	if handler == nil {
		return fmt.Errorf("%w: sse handler is nil", ErrRouteNotRegistered)
	}
	return nil
}

func newSSEOperation[I any](
	method string,
	registerPath string,
	fullPath string,
	operationOptions []OperationOption,
	policies []SSERoutePolicy[I],
) huma.Operation {
	op := huma.Operation{
		OperationID: defaultOperationID(method, fullPath),
		Method:      method,
		Path:        registerPath,
	}
	option.Apply(&op, operationOptions...)
	applySSEPolicyOperations(&op, policies)
	return op
}
