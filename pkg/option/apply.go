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