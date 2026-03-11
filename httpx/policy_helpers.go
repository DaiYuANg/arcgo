package httpx

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/samber/lo"
)

func buildOperationMutation(operationOptions []OperationOption) func(*huma.Operation) {
	return func(op *huma.Operation) {
		if op == nil {
			return
		}
		lo.ForEach(operationOptions, func(opt OperationOption, _ int) {
			if opt != nil {
				opt(op)
			}
		})
	}
}

func applyOperationMutations(op *huma.Operation, mutators []func(*huma.Operation)) {
	if op == nil || len(mutators) == 0 {
		return
	}
	lo.ForEach(mutators, func(mutate func(*huma.Operation), _ int) {
		if mutate != nil {
			mutate(op)
		}
	})
}

func applyWrappers[T any](handler T, wrappers []func(T) T) T {
	wrapped := handler
	for i := len(wrappers) - 1; i >= 0; i-- {
		if wrappers[i] != nil {
			wrapped = wrappers[i](wrapped)
		}
	}
	return wrapped
}
