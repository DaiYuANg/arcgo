package option

// Apply executes non-nil option functions against the target in order.
func Apply[T any, O ~func(*T)](target *T, opts ...O) {
	if target == nil || len(opts) == 0 {
		return
	}

	for _, opt := range opts {
		if opt != nil {
			opt(target)
		}
	}
}
