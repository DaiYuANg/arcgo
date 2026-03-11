package httpx

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/samber/lo"
)

// Route registers a typed handler on the server.
func Route[I, O any](s ServerRuntime, method, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	server := unwrapServer(s)
	if server == nil {
		return fmt.Errorf("%w: server is nil", ErrRouteNotRegistered)
	}
	fullPath := joinRoutePath(server.basePath, path)
	return registerTyped(server, server.HumaAPI(), method, fullPath, fullPath, handler, operationOptions, nil)
}

// Get registers a typed GET handler on the server.
func Get[I, O any](s ServerRuntime, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return Route(s, MethodGet, path, handler, operationOptions...)
}

// Post registers a typed POST handler on the server.
func Post[I, O any](s ServerRuntime, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return Route(s, MethodPost, path, handler, operationOptions...)
}

// Put registers a typed PUT handler on the server.
func Put[I, O any](s ServerRuntime, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return Route(s, MethodPut, path, handler, operationOptions...)
}

// Patch registers a typed PATCH handler on the server.
func Patch[I, O any](s ServerRuntime, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return Route(s, MethodPatch, path, handler, operationOptions...)
}

// Delete registers a typed DELETE handler on the server.
func Delete[I, O any](s ServerRuntime, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return Route(s, MethodDelete, path, handler, operationOptions...)
}

// Head registers a typed HEAD handler on the server.
func Head[I, O any](s ServerRuntime, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return Route(s, MethodHead, path, handler, operationOptions...)
}

// Options registers a typed OPTIONS handler on the server.
func Options[I, O any](s ServerRuntime, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return Route(s, MethodOptions, path, handler, operationOptions...)
}

// GroupRoute registers a typed handler under a route group.
func GroupRoute[I, O any](g *Group, method, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
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

	return registerTyped(g.server, target, method, registerPath, fullPath, handler, operationOptions, nil)
}

// GroupGet registers a typed GET handler under a route group.
func GroupGet[I, O any](g *Group, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return GroupRoute(g, MethodGet, path, handler, operationOptions...)
}

// GroupPost registers a typed POST handler under a route group.
func GroupPost[I, O any](g *Group, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return GroupRoute(g, MethodPost, path, handler, operationOptions...)
}

// GroupPut registers a typed PUT handler under a route group.
func GroupPut[I, O any](g *Group, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return GroupRoute(g, MethodPut, path, handler, operationOptions...)
}

// GroupPatch registers a typed PATCH handler under a route group.
func GroupPatch[I, O any](g *Group, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return GroupRoute(g, MethodPatch, path, handler, operationOptions...)
}

// GroupDelete registers a typed DELETE handler under a route group.
func GroupDelete[I, O any](g *Group, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return GroupRoute(g, MethodDelete, path, handler, operationOptions...)
}

// MustRoute registers a route and panics if registration fails.
func MustRoute[I, O any](s ServerRuntime, method, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) {
	lo.Must0(Route(s, method, path, handler, operationOptions...))
}

// MustGet registers a GET route and panics if registration fails.
func MustGet[I, O any](s ServerRuntime, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) {
	lo.Must0(Get(s, path, handler, operationOptions...))
}

// MustPost registers a POST route and panics if registration fails.
func MustPost[I, O any](s ServerRuntime, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) {
	lo.Must0(Post(s, path, handler, operationOptions...))
}

// MustPut registers a PUT route and panics if registration fails.
func MustPut[I, O any](s ServerRuntime, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) {
	lo.Must0(Put(s, path, handler, operationOptions...))
}

// MustPatch registers a PATCH route and panics if registration fails.
func MustPatch[I, O any](s ServerRuntime, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) {
	lo.Must0(Patch(s, path, handler, operationOptions...))
}

// MustDelete registers a DELETE route and panics if registration fails.
func MustDelete[I, O any](s ServerRuntime, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) {
	lo.Must0(Delete(s, path, handler, operationOptions...))
}

// MustHead registers a HEAD route and panics if registration fails.
func MustHead[I, O any](s ServerRuntime, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) {
	lo.Must0(Head(s, path, handler, operationOptions...))
}

// MustOptions registers an OPTIONS route and panics if registration fails.
func MustOptions[I, O any](s ServerRuntime, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) {
	lo.Must0(Options(s, path, handler, operationOptions...))
}

// MustGroupRoute registers a grouped route and panics if registration fails.
func MustGroupRoute[I, O any](g *Group, method, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) {
	lo.Must0(GroupRoute(g, method, path, handler, operationOptions...))
}

// MustGroupGet registers a grouped GET route and panics if registration fails.
func MustGroupGet[I, O any](g *Group, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) {
	lo.Must0(GroupGet(g, path, handler, operationOptions...))
}

// MustGroupPost registers a grouped POST route and panics if registration fails.
func MustGroupPost[I, O any](g *Group, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) {
	lo.Must0(GroupPost(g, path, handler, operationOptions...))
}

// MustGroupPut registers a grouped PUT route and panics if registration fails.
func MustGroupPut[I, O any](g *Group, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) {
	lo.Must0(GroupPut(g, path, handler, operationOptions...))
}

// MustGroupPatch registers a grouped PATCH route and panics if registration fails.
func MustGroupPatch[I, O any](g *Group, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) {
	lo.Must0(GroupPatch(g, path, handler, operationOptions...))
}

// MustGroupDelete registers a grouped DELETE route and panics if registration fails.
func MustGroupDelete[I, O any](g *Group, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) {
	lo.Must0(GroupDelete(g, path, handler, operationOptions...))
}

// registerTyped converts an httpx typed handler into a Huma operation registration.
func registerTyped[I, O any](
	s *Server,
	api huma.API,
	method, registerPath, fullPath string,
	handler TypedHandler[I, O],
	operationOptions []OperationOption,
	policies []RoutePolicy[I, O],
) error {
	if s == nil {
		return fmt.Errorf("%w: server is nil", ErrRouteNotRegistered)
	}
	if api == nil {
		return ErrAdapterNotFound
	}

	wrappedHandler := withInputValidation(s, handler)
	wrappedHandler = applyRoutePolicies(wrappedHandler, policies)

	opID := defaultOperationID(method, fullPath)
	handlerName := handlerName(handler)

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
	applyPolicyOperations(&op, policies)

	lo.ForEach(s.operationModifiers.Values(), func(modifier func(*huma.Operation), _ int) {
		if modifier != nil {
			modifier(&op)
		}
	})

	s.openAPIMu.Lock()
	defer s.openAPIMu.Unlock()
	huma.Register(api, op, func(ctx context.Context, input *I) (*O, error) {
		return wrappedHandler(ctx, input)
	})

	s.addRoute(RouteInfo{
		Method:      method,
		Path:        fullPath,
		HandlerName: handlerName,
	})

	return nil
}

// withInputValidation applies validator checks and standard error conversion.
func withInputValidation[I, O any](s *Server, handler TypedHandler[I, O]) TypedHandler[I, O] {
	if handler == nil || s == nil {
		return handler
	}

	validateInput := compileInputValidator[I](s.validator)

	return func(ctx context.Context, input *I) (out *O, err error) {
		if s.panicRecover {
			defer func() {
				if recovered := recover(); recovered != nil {
					out = nil
					err = huma.Error500InternalServerError(fmt.Sprintf("panic in handler: %v", recovered))
				}
			}()
		}

		if validateInput != nil {
			if err = validateInput(input); err != nil {
				message := validationErrorMessage(err)
				return nil, huma.Error400BadRequest(message, err)
			}
		}

		out, err = handler(ctx, input)
		if err != nil {
			if httpxErr, ok := errors.AsType[*Error](err); ok {
				return nil, lo.Ternary(
					httpxErr.Err != nil,
					huma.NewError(httpxErr.Code, httpxErr.Message, httpxErr.Err),
					huma.NewError(httpxErr.Code, httpxErr.Message),
				)
			}

			if _, ok := errors.AsType[huma.StatusError](err); ok {
				return nil, err
			}

			return nil, huma.Error500InternalServerError(err.Error(), err)
		}
		return out, nil
	}
}

// handlerName returns a best-effort function name for diagnostics.
func handlerName(fn any) string {
	v := reflect.ValueOf(fn)
	if !v.IsValid() || v.Kind() != reflect.Func {
		return "unknown"
	}

	runtimeFn := runtime.FuncForPC(v.Pointer())
	return lo.Ternary(runtimeFn != nil, lo.LastOr(strings.Split(runtimeFn.Name(), "/"), "unknown"), "unknown")
}

// defaultOperationID generates a stable fallback operation id from method and path.
func defaultOperationID(method, path string) string {
	cleanPath := strings.Trim(path, "/")
	cleanPath = lo.Ternary(cleanPath == "", "root", cleanPath)
	cleanPath = strings.NewReplacer("/", "-", "{", "", "}", "", ":", "").Replace(cleanPath)
	return strings.ToLower(method) + "-" + cleanPath
}
