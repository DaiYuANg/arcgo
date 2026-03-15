package httpx

import (
	"context"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/DaiYuANg/archgo/collectionx/set"
	"github.com/danielgtaylor/huma/v2"
	"github.com/samber/lo"
)

// OperationBinaryResponse documents a binary payload for HTTP 200.
func OperationBinaryResponse(contentTypes ...string) OperationOption {
	normalized := normalizeContentTypes(contentTypes, "application/octet-stream")

	return func(op *huma.Operation) {
		if op == nil {
			return
		}

		code := strconv.Itoa(http.StatusOK)
		if op.Responses == nil {
			op.Responses = map[string]*huma.Response{}
		}
		if op.Responses[code] == nil {
			op.Responses[code] = &huma.Response{
				Description: http.StatusText(http.StatusOK),
			}
		}
		if op.Responses[code].Content == nil {
			op.Responses[code].Content = map[string]*huma.MediaType{}
		}

		for _, contentType := range normalized {
			if _, exists := op.Responses[code].Content[contentType]; exists {
				continue
			}
			op.Responses[code].Content[contentType] = &huma.MediaType{
				Schema: &huma.Schema{
					Type:   huma.TypeString,
					Format: "binary",
				},
			}
		}
	}
}

// OperationHTMLResponse documents HTML payload for HTTP 200.
func OperationHTMLResponse() OperationOption {
	return func(op *huma.Operation) {
		if op == nil {
			return
		}

		code := strconv.Itoa(http.StatusOK)
		if op.Responses == nil {
			op.Responses = map[string]*huma.Response{}
		}
		if op.Responses[code] == nil {
			op.Responses[code] = &huma.Response{
				Description: http.StatusText(http.StatusOK),
			}
		}
		if op.Responses[code].Content == nil {
			op.Responses[code].Content = map[string]*huma.MediaType{}
		}
		if _, exists := op.Responses[code].Content["text/html"]; exists {
			return
		}
		op.Responses[code].Content["text/html"] = &huma.MediaType{
			Schema: &huma.Schema{
				Type: huma.TypeString,
			},
		}
	}
}

// PolicyImageResponse applies runtime default Content-Type and OpenAPI binary response.
func PolicyImageResponse[I, O any](contentTypes ...string) RoutePolicy[I, O] {
	normalized := normalizeContentTypes(contentTypes, "image/png")
	defaultType := normalized[0]
	headerSetter := compileHeaderSetter[O]("Content-Type", defaultType)

	return RoutePolicy[I, O]{
		Name:      "image-response",
		Operation: OperationBinaryResponse(normalized...),
		Wrap: func(next TypedHandler[I, O]) TypedHandler[I, O] {
			if next == nil {
				return nil
			}
			return func(ctx context.Context, input *I) (*O, error) {
				out, err := next(ctx, input)
				if err != nil || out == nil {
					return out, err
				}
				if headerSetter != nil {
					headerSetter(out)
				}
				return out, nil
			}
		},
	}
}

// PolicyHTMLResponse applies runtime default Content-Type and OpenAPI HTML response.
func PolicyHTMLResponse[I, O any]() RoutePolicy[I, O] {
	headerSetter := compileHeaderSetter[O]("Content-Type", "text/html")

	return RoutePolicy[I, O]{
		Name:      "html-response",
		Operation: OperationHTMLResponse(),
		Wrap: func(next TypedHandler[I, O]) TypedHandler[I, O] {
			if next == nil {
				return nil
			}
			return func(ctx context.Context, input *I) (*O, error) {
				out, err := next(ctx, input)
				if err != nil || out == nil {
					return out, err
				}
				if headerSetter != nil {
					headerSetter(out)
				}
				return out, nil
			}
		},
	}
}

func compileHeaderSetter[O any](headerName, headerValue string) func(*O) {
	if headerName == "" || headerValue == "" {
		return nil
	}

	outputType := reflect.TypeFor[O]()
	for outputType.Kind() == reflect.Pointer {
		outputType = outputType.Elem()
	}
	if outputType.Kind() != reflect.Struct {
		return nil
	}

	fieldIndex := -1
	for i := 0; i < outputType.NumField(); i++ {
		structField := outputType.Field(i)
		if !strings.EqualFold(structField.Tag.Get("header"), headerName) {
			continue
		}
		if structField.Type.Kind() == reflect.String {
			fieldIndex = i
			break
		}
	}
	if fieldIndex < 0 {
		return nil
	}

	return func(output *O) {
		if output == nil {
			return
		}

		value := reflect.ValueOf(output)
		if !value.IsValid() || value.IsNil() {
			return
		}

		value = value.Elem()
		for value.IsValid() && value.Kind() == reflect.Pointer {
			if value.IsNil() {
				return
			}
			value = value.Elem()
		}
		if !value.IsValid() || value.Kind() != reflect.Struct || fieldIndex >= value.NumField() {
			return
		}

		field := value.Field(fieldIndex)
		if field.Kind() == reflect.String && field.CanSet() && field.String() == "" {
			field.SetString(headerValue)
		}
	}
}

func normalizeContentTypes(values []string, fallback string) []string {
	if len(values) == 0 {
		return []string{fallback}
	}

	ordered := set.NewOrderedSet[string]()
	lo.ForEach(lo.FilterMap(values, func(value string, _ int) (string, bool) {
		trimmed := strings.TrimSpace(value)
		return trimmed, trimmed != ""
	}), func(contentType string, _ int) {
		ordered.Add(contentType)
	})
	normalized := ordered.Values()
	if len(normalized) == 0 {
		return []string{fallback}
	}
	return normalized
}
