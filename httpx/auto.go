package httpx

import (
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/casing"
	"github.com/samber/lo"
	"github.com/samber/oops"
)

// AutoRoute is a preconfigured auto-inferred route registration unit.
type AutoRoute interface {
	registerAuto(Registrar) error
}

type autoRoute[I, O any] struct {
	handler          TypedHandler[I, O]
	operationOptions []OperationOption
}

type autoVerbSpec struct {
	Name    string
	Method  string
	Summary string
}

type autoRoutePattern struct {
	HandlerName string
	Method      string
	Path        string
	Summary     string
}

var autoVerbSpecs = []autoVerbSpec{
	{Name: "Options", Method: MethodOptions, Summary: "Options"},
	{Name: "Delete", Method: MethodDelete, Summary: "Delete"},
	{Name: "Create", Method: MethodPost, Summary: "Create"},
	{Name: "Update", Method: MethodPut, Summary: "Update"},
	{Name: "Patch", Method: MethodPatch, Summary: "Patch"},
	{Name: "Head", Method: MethodHead, Summary: "Head"},
	{Name: "List", Method: MethodGet, Summary: "List"},
	{Name: "Get", Method: MethodGet, Summary: "Get"},
}

// Auto builds an auto-inferred route registration from a typed handler method.
// The handler name determines the HTTP method and scoped path.
func Auto[I, O any](handler TypedHandler[I, O], operationOptions ...OperationOption) AutoRoute {
	return autoRoute[I, O]{
		handler:          handler,
		operationOptions: operationOptions,
	}
}

// RegisterAuto registers one or more auto-inferred routes on the provided registrar.
func RegisterAuto(registrar Registrar, routes ...AutoRoute) error {
	if registrar == nil {
		return oops.In("httpx").
			With("op", "register_auto_routes", "route_count", len(routes)).
			Wrapf(ErrRouteNotRegistered, "validate registrar")
	}

	for _, route := range routes {
		if route == nil {
			continue
		}
		if err := route.registerAuto(registrar); err != nil {
			return err
		}
	}

	return nil
}

// MustAuto registers auto-inferred routes and panics if registration fails.
func MustAuto(registrar Registrar, routes ...AutoRoute) {
	lo.Must0(RegisterAuto(registrar, routes...))
}

func (r autoRoute[I, O]) registerAuto(registrar Registrar) error {
	pattern, err := inferAutoRoutePattern(r.handler)
	if err != nil {
		return oops.In("httpx").
			With("op", "register_auto_route").
			Wrapf(err, "infer auto route")
	}

	operationOptions := append([]OperationOption{
		func(op *huma.Operation) {
			if strings.TrimSpace(op.Summary) == "" {
				op.Summary = pattern.Summary
			}
		},
	}, r.operationOptions...)

	return GroupRoute(registrar.Scope(), pattern.Method, pattern.Path, r.handler, operationOptions...)
}

func inferAutoRoutePattern(handler any) (autoRoutePattern, error) {
	name, err := autoHandlerMethodName(handler)
	if err != nil {
		return autoRoutePattern{}, err
	}

	for _, verb := range autoVerbSpecs {
		if !strings.HasPrefix(name, verb.Name) {
			continue
		}

		remainder := strings.TrimPrefix(name, verb.Name)
		resourcePart, paramParts, err := splitAutoRouteTokens(remainder)
		if err != nil {
			return autoRoutePattern{}, err
		}

		path, err := buildAutoRoutePath(resourcePart, paramParts)
		if err != nil {
			return autoRoutePattern{}, err
		}

		return autoRoutePattern{
			HandlerName: name,
			Method:      verb.Method,
			Path:        path,
			Summary:     buildAutoRouteSummary(verb.Summary, resourcePart),
		}, nil
	}

	return autoRoutePattern{}, oops.In("httpx").
		With("handler_name", name).
		Wrapf(ErrInvalidHandlerName, "resolve auto route verb")
}

func autoHandlerMethodName(handler any) (string, error) {
	fullName := handlerName(handler)
	if fullName == "unknown" {
		return "", oops.In("httpx").
			Wrapf(ErrInvalidHandlerName, "resolve handler runtime name")
	}

	shortName := fullName
	if lastDot := strings.LastIndex(shortName, "."); lastDot >= 0 && lastDot < len(shortName)-1 {
		shortName = shortName[lastDot+1:]
	}
	shortName = strings.TrimSuffix(shortName, "-fm")
	if shortName == "" || strings.HasPrefix(shortName, "func") {
		return "", oops.In("httpx").
			With("handler_name", fullName).
			Wrapf(ErrInvalidHandlerName, "validate handler method name")
	}

	return shortName, nil
}

func splitAutoRouteTokens(remainder string) (string, []string, error) {
	if strings.TrimSpace(remainder) == "" {
		return "", nil, nil
	}

	tokens := casing.Split(remainder)
	if len(tokens) == 0 {
		return "", nil, nil
	}

	byIndex := -1
	for i, token := range tokens {
		if token == "By" {
			byIndex = i
			break
		}
	}

	if byIndex < 0 {
		return strings.Join(tokens, ""), nil, nil
	}
	if byIndex == len(tokens)-1 {
		return "", nil, oops.In("httpx").
			With("remainder", remainder).
			Wrapf(ErrInvalidHandlerName, "validate auto route params")
	}

	resourcePart := strings.Join(tokens[:byIndex], "")
	paramTokens := tokens[byIndex+1:]
	parts := make([]string, 0, 2)
	current := make([]string, 0, 2)
	for _, token := range paramTokens {
		if token == "And" {
			if len(current) == 0 {
				return "", nil, oops.In("httpx").
					With("remainder", remainder).
					Wrapf(ErrInvalidHandlerName, "validate auto route params")
			}
			parts = append(parts, strings.Join(current, ""))
			current = current[:0]
			continue
		}
		current = append(current, token)
	}
	if len(current) == 0 {
		return "", nil, oops.In("httpx").
			With("remainder", remainder).
			Wrapf(ErrInvalidHandlerName, "validate auto route params")
	}
	parts = append(parts, strings.Join(current, ""))

	return resourcePart, parts, nil
}

func buildAutoRoutePath(resourcePart string, paramParts []string) (string, error) {
	segments := make([]string, 0, 4)
	if strings.TrimSpace(resourcePart) != "" {
		segments = append(segments, casing.Kebab(resourcePart))
	}

	if len(paramParts) > 0 {
		for _, part := range paramParts {
			if strings.TrimSpace(part) == "" {
				return "", oops.In("httpx").
					With("resource_part", resourcePart).
					Wrapf(ErrInvalidHandlerName, "validate auto route params")
			}
			segments = append(segments, "{"+casing.Kebab(part)+"}")
		}
	}

	if len(segments) == 0 {
		return "", nil
	}

	return "/" + strings.Join(segments, "/"), nil
}

func buildAutoRouteSummary(verbSummary, resourcePart string) string {
	if strings.TrimSpace(resourcePart) == "" {
		return verbSummary
	}

	return verbSummary + " " + casing.Join(casing.Split(resourcePart), " ", strings.ToLower, strings.Title, casing.Initialism) //nolint:staticcheck
}
