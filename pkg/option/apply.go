package option

import "github.com/samber/lo"

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

	for _, opt := range lo.Filter(opts, func(opt O, _ int) bool { return opt != nil }) {
		if err := opt(target); err != nil {
			return err
		}
	}

	return nil
}
