package httpx

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"strings"

	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/danielgtaylor/huma/v2"
)

// Route 注册泛型强类型路由。
func Route[I, O any](s *Server, method, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return registerTypedWithPrefix(s, "", method, path, handler, operationOptions...)
}

// Get 注册 GET 路由。
func Get[I, O any](s *Server, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return Route(s, http.MethodGet, path, handler, operationOptions...)
}

// Post 注册 POST 路由。
func Post[I, O any](s *Server, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return Route(s, http.MethodPost, path, handler, operationOptions...)
}

// Put 注册 PUT 路由。
func Put[I, O any](s *Server, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return Route(s, http.MethodPut, path, handler, operationOptions...)
}

// Patch 注册 PATCH 路由。
func Patch[I, O any](s *Server, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return Route(s, http.MethodPatch, path, handler, operationOptions...)
}

// Delete 注册 DELETE 路由。
func Delete[I, O any](s *Server, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return Route(s, http.MethodDelete, path, handler, operationOptions...)
}

// Head 注册 HEAD 路由。
func Head[I, O any](s *Server, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return Route(s, http.MethodHead, path, handler, operationOptions...)
}

// Options 注册 OPTIONS 路由。
func Options[I, O any](s *Server, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return Route(s, http.MethodOptions, path, handler, operationOptions...)
}

// GroupRoute 在分组下注册泛型路由。
func GroupRoute[I, O any](g *Group, method, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	if g == nil || g.server == nil {
		return fmt.Errorf("%w: route group is nil", ErrRouteNotRegistered)
	}
	return registerTypedWithPrefix(g.server, g.prefix, method, path, handler, operationOptions...)
}

// GroupGet 在分组下注册 GET 路由。
func GroupGet[I, O any](g *Group, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return GroupRoute(g, http.MethodGet, path, handler, operationOptions...)
}

// GroupPost 在分组下注册 POST 路由。
func GroupPost[I, O any](g *Group, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return GroupRoute(g, http.MethodPost, path, handler, operationOptions...)
}

// GroupPut 在分组下注册 PUT 路由。
func GroupPut[I, O any](g *Group, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return GroupRoute(g, http.MethodPut, path, handler, operationOptions...)
}

// GroupPatch 在分组下注册 PATCH 路由。
func GroupPatch[I, O any](g *Group, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return GroupRoute(g, http.MethodPatch, path, handler, operationOptions...)
}

// GroupDelete 在分组下注册 DELETE 路由。
func GroupDelete[I, O any](g *Group, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	return GroupRoute(g, http.MethodDelete, path, handler, operationOptions...)
}

func registerTypedWithPrefix[I, O any](s *Server, prefix, method, path string, handler TypedHandler[I, O], operationOptions ...OperationOption) error {
	if s == nil {
		return fmt.Errorf("%w: server is nil", ErrRouteNotRegistered)
	}
	if handler == nil {
		return fmt.Errorf("%w: handler cannot be nil", ErrRouteNotRegistered)
	}

	fullPath := joinRoutePath(joinRoutePath(s.basePath, prefix), path)
	upperMethod := strings.ToUpper(strings.TrimSpace(method))
	if upperMethod == "" {
		return fmt.Errorf("%w: method is required", ErrRouteNotRegistered)
	}

	op := huma.Operation{
		OperationID: defaultOperationID(upperMethod, fullPath),
		Method:      upperMethod,
		Path:        fullPath,
	}
	for _, apply := range operationOptions {
		if apply != nil {
			apply(&op)
		}
	}

	if s.humaOpts.Enabled {
		api := s.adapter.HumaAPI()
		if api == nil {
			return fmt.Errorf("%w: adapter %q does not expose Huma API", ErrAdapterNotFound, s.adapter.Name())
		}
		huma.Register[I, O](api, op, handler)
	} else {
		s.adapter.Handle(upperMethod, fullPath, wrapTypedHandler(s, handler))
	}

	s.routes = append(s.routes, RouteInfo{
		Method:      upperMethod,
		Path:        fullPath,
		HandlerName: handlerName(handler),
		Comment:     op.Summary,
		Tags:        op.Tags,
	})

	s.printRoutesIfEnabled()
	return nil
}

func wrapTypedHandler[I, O any](s *Server, handler TypedHandler[I, O]) adapter.HandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.writeHandlerError(w, r, NewError(http.StatusInternalServerError, fmt.Sprintf("panic in handler: %v", recovered)))
			}
		}()

		in := new(I)
		if err := bindRequestToInput(r, in); err != nil {
			s.writeHandlerError(w, r, NewError(http.StatusBadRequest, "invalid request input", err))
			return nil
		}

		out, err := handler(ctx, in)
		if err != nil {
			s.writeHandlerError(w, r, err)
			return nil
		}

		if out == nil {
			w.WriteHeader(http.StatusNoContent)
			return nil
		}

		status := statusCodeFrom(out)
		writeJSON(w, status, out)
		return nil
	}
}

func statusCodeFrom(v any) int {
	type statusCoder interface {
		StatusCode() int
	}
	if sc, ok := v.(statusCoder); ok {
		code := sc.StatusCode()
		if code > 0 {
			return code
		}
	}
	return http.StatusOK
}

func shouldDecodeBody(r *http.Request) bool {
	if r == nil || r.Body == nil || r.Body == http.NoBody {
		return false
	}

	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodDelete, http.MethodOptions:
		return false
	default:
		return true
	}
}

func handlerName(fn any) string {
	v := reflect.ValueOf(fn)
	if !v.IsValid() || v.Kind() != reflect.Func {
		return "unknown"
	}
	if runtimeFn := runtime.FuncForPC(v.Pointer()); runtimeFn != nil {
		parts := strings.Split(runtimeFn.Name(), "/")
		return parts[len(parts)-1]
	}
	return "unknown"
}

func defaultOperationID(method, path string) string {
	cleanPath := strings.Trim(path, "/")
	if cleanPath == "" {
		cleanPath = "root"
	}
	cleanPath = strings.NewReplacer("/", "-", "{", "", "}", "", ":", "").Replace(cleanPath)
	return strings.ToLower(method) + "-" + cleanPath
}
