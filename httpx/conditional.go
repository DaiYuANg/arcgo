package httpx

import (
	"context"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

// ConditionalStateGetter resolves a resource state used by conditional request checks.
type ConditionalStateGetter[I any] func(ctx context.Context, input *I) (etag string, modified time.Time, err error)

// OperationConditionalRead documents HTTP 304 for conditional read requests.
func OperationConditionalRead() OperationOption {
	return operationConditionalResponse(http.StatusNotModified)
}

// OperationConditionalWrite documents HTTP 412 for conditional write requests.
func OperationConditionalWrite() OperationOption {
	return operationConditionalResponse(http.StatusPreconditionFailed)
}

// PolicyConditionalRead checks conditional headers and documents HTTP 304.
func PolicyConditionalRead[I, O any](stateGetter ConditionalStateGetter[I]) RoutePolicy[I, O] {
	return conditionalPolicy[I, O](OperationConditionalRead(), stateGetter)
}

// PolicyConditionalWrite checks conditional headers and documents HTTP 412.
func PolicyConditionalWrite[I, O any](stateGetter ConditionalStateGetter[I]) RoutePolicy[I, O] {
	return conditionalPolicy[I, O](OperationConditionalWrite(), stateGetter)
}

func conditionalPolicy[I, O any](operationOption OperationOption, stateGetter ConditionalStateGetter[I]) RoutePolicy[I, O] {
	paramsExtractor := compileConditionalParamsExtractor[I]()

	return RoutePolicy[I, O]{
		Name:      "conditional",
		Operation: operationOption,
		Wrap: func(next TypedHandler[I, O]) TypedHandler[I, O] {
			if next == nil || stateGetter == nil || paramsExtractor == nil {
				return next
			}
			return func(ctx context.Context, input *I) (*O, error) {
				params := paramsExtractor(input)
				if params == nil || !params.HasConditionalParams() {
					return next(ctx, input)
				}

				etag, modified, err := stateGetter(ctx, input)
				if err != nil {
					return nil, err
				}
				if err := params.PreconditionFailed(etag, modified); err != nil {
					return nil, err
				}
				return next(ctx, input)
			}
		},
	}
}

func operationConditionalResponse(status int) OperationOption {
	return func(op *huma.Operation) {
		if op == nil {
			return
		}
		if op.Responses == nil {
			op.Responses = map[string]*huma.Response{}
		}

		code := strconv.Itoa(status)
		if _, exists := op.Responses[code]; exists {
			return
		}

		op.Responses[code] = &huma.Response{
			Description: http.StatusText(status),
		}
	}
}

func compileConditionalParamsExtractor[I any]() func(*I) *ConditionalParams {
	inputType := reflect.TypeFor[I]()
	for inputType.Kind() == reflect.Pointer {
		inputType = inputType.Elem()
	}
	if inputType.Kind() != reflect.Struct {
		return nil
	}

	paramsType := reflect.TypeFor[ConditionalParams]()
	paramsPtrType := reflect.PointerTo(paramsType)

	fieldIndex := -1
	isPointerField := false
	for i := 0; i < inputType.NumField(); i++ {
		fieldType := inputType.Field(i).Type
		switch fieldType {
		case paramsType:
			fieldIndex = i
		case paramsPtrType:
			fieldIndex = i
			isPointerField = true
		}
		if fieldIndex >= 0 {
			break
		}
	}
	if fieldIndex < 0 {
		return nil
	}

	return func(input *I) *ConditionalParams {
		if input == nil {
			return nil
		}

		value := reflect.ValueOf(input)
		if !value.IsValid() || value.IsNil() {
			return nil
		}

		value = value.Elem()
		for value.IsValid() && value.Kind() == reflect.Pointer {
			if value.IsNil() {
				return nil
			}
			value = value.Elem()
		}
		if !value.IsValid() || value.Kind() != reflect.Struct || fieldIndex >= value.NumField() {
			return nil
		}

		field := value.Field(fieldIndex)
		if isPointerField {
			if field.IsNil() || !field.CanInterface() {
				return nil
			}
			params, _ := field.Interface().(*ConditionalParams)
			return params
		}

		if !field.CanAddr() {
			return nil
		}
		addr := field.Addr()
		if !addr.CanInterface() {
			return nil
		}
		params, _ := addr.Interface().(*ConditionalParams)
		return params
	}

}
