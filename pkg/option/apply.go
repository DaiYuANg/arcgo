package option

import (
	"fmt"

	"github.com/samber/lo"
)

// Apply executes non-nil option functions against the target in order.
func Apply[T any, O ~func(*T)](target *T, opts ...O) {
	if target == nil || len(opts) == 0 {
		return
	}

	lo.ForEach(opts, func(opt O, _ int) {
		if opt != nil {
			opt(target)
		}
	})
}

// ApplyErr executes non-nil option functions against the target in order and stops on the first error.
func ApplyErr[T any, O ~func(*T) error](target *T, opts ...O) error {
	if target == nil || len(opts) == 0 {
		return nil
	}

	_, err := lo.ReduceErr(lo.Filter(opts, func(opt O, _ int) bool {
		return opt != nil
	}), func(_ struct{}, opt O, _ int) (struct{}, error) {
		return struct{}{}, opt(target)
	}, struct{}{})
	if err != nil {
		return fmt.Errorf("apply options: %w", err)
	}
	return nil
}
