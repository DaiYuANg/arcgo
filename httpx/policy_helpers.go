package httpx

import (
	"github.com/DaiYuANg/arcgo/pkg/option"
	"github.com/danielgtaylor/huma/v2"
	"github.com/samber/lo"
)

func buildOperationMutation(operationOptions []OperationOption) func(*huma.Operation) {
	return func(op *huma.Operation) {
		if op == nil {
			return
		}
		option.Apply(op, operationOptions...)
	}
}

func applyOperationMutations(op *huma.Operation, mutators []func(*huma.Operation)) {
	if op == nil || len(mutators) == 0 {
		return
	}
	option.Apply(op, mutators...)
}

func applyWrappers[T any](handler T, wrappers []func(T) T) T {
	return lo.ReduceRight(wrappers, func(wrapped T, wrapper func(T) T, _ int) T {
		if wrapper == nil {
			return wrapped
		}
		return wrapper(wrapped)
	}, handler)
}
