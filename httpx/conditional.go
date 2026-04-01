package httpx

import (
	"context"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/samber/lo"
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
		Wrap:      nextConditionalHandler[I, O](stateGetter, paramsExtractor),
	}
}

func nextConditionalHandler[I, O any](
	stateGetter ConditionalStateGetter[I],
	paramsExtractor func(*I) *ConditionalParams,
) func(next TypedHandler[I, O]) TypedHandler[I, O] {
	if stateGetter == nil || paramsExtractor == nil {
		return nil
	}

	return func(next TypedHandler[I, O]) TypedHandler[I, O] {
		if next == nil {
			return nil
		}
		return conditionalRequestHandler(next, stateGetter, paramsExtractor)
	}
}

func conditionalRequestHandler[I, O any](
	next TypedHandler[I, O],
	stateGetter ConditionalStateGetter[I],
	paramsExtractor func(*I) *ConditionalParams,
) TypedHandler[I, O] {
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
	inputType, _, ok := indirectStructType[I]()
	if !ok {
		return nil
	}

	fieldIndex, isPointerField, ok := conditionalParamsField(inputType)
	if !ok {
		return nil
	}

	return func(input *I) *ConditionalParams {
		return extractConditionalParams(input, fieldIndex, isPointerField)
	}
}

func conditionalParamsField(inputType reflect.Type) (int, bool, bool) {
	paramsType := reflect.TypeFor[ConditionalParams]()
	paramsPtrType := reflect.PointerTo(paramsType)

	index, ok := lo.Find(lo.Range(inputType.NumField()), func(index int) bool {
		fieldType := inputType.Field(index).Type
		return fieldType == paramsType || fieldType == paramsPtrType
	})
	if !ok {
		return 0, false, false
	}

	return index, inputType.Field(index).Type == paramsPtrType, true
}

func extractConditionalParams[I any](input *I, fieldIndex int, isPointerField bool) *ConditionalParams {
	value, hasValue := indirectStructValue(input)
	if !hasValue || fieldIndex >= value.NumField() {
		return nil
	}

	field := value.Field(fieldIndex)
	if isPointerField {
		return conditionalParamsFromPointerField(field)
	}
	return conditionalParamsFromValueField(field)
}

func conditionalParamsFromPointerField(field reflect.Value) *ConditionalParams {
	if field.IsNil() || !field.CanInterface() {
		return nil
	}
	params, ok := field.Interface().(*ConditionalParams)
	if !ok {
		return nil
	}
	return params
}

func conditionalParamsFromValueField(field reflect.Value) *ConditionalParams {
	if !field.CanAddr() {
		return nil
	}
	addr := field.Addr()
	if !addr.CanInterface() {
		return nil
	}
	params, ok := addr.Interface().(*ConditionalParams)
	if !ok {
		return nil
	}
	return params
}
