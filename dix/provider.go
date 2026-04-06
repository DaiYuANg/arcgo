package dix

// ProviderFunc describes a typed provider registration.
type ProviderFunc struct {
	register func(*Container)
	meta     ProviderMetadata
}

func (p ProviderFunc) apply(c *Container) {
	if p.register != nil {
		p.register(c)
	}
}

// RawProvider registers an untyped provider callback.
func RawProvider(fn func(*Container)) ProviderFunc {
	return RawProviderWithMetadata(fn, ProviderMetadata{
		Label: "RawProvider",
	})
}

// RawProviderWithMetadata registers an untyped provider callback with metadata.
func RawProviderWithMetadata(fn func(*Container), meta ProviderMetadata) ProviderFunc {
	return NewProviderFunc(fn, ProviderMetadata{
		Label:        meta.Label,
		Output:       meta.Output,
		Dependencies: meta.Dependencies,
		Raw:          true,
	})
}

// Provider0 registers a typed singleton provider with no dependencies.
func Provider0[T any](fn func() T) ProviderFunc {
	return NewProviderFunc(
		func(c *Container) { ProvideT(c, fn) },
		ProviderMetadata{
			Label:  "Provider0",
			Output: TypedService[T](),
		},
	)
}

// ProviderErr0 registers a typed singleton provider with no dependencies.
func ProviderErr0[T any](fn func() (T, error)) ProviderFunc {
	return NewProviderFunc(
		func(c *Container) { ProvideTErr(c, fn) },
		ProviderMetadata{
			Label:  "ProviderErr0",
			Output: TypedService[T](),
		},
	)
}

// Provider1 registers a typed singleton provider with one dependency.
func Provider1[T, D1 any](fn func(D1) T) ProviderFunc {
	return NewProviderFunc(
		func(c *Container) { Provide1T(c, fn) },
		ProviderMetadata{
			Label:        "Provider1",
			Output:       TypedService[T](),
			Dependencies: ServiceRefs(TypedService[D1]()),
		},
	)
}

// ProviderErr1 registers a typed singleton provider with one dependency.
func ProviderErr1[T, D1 any](fn func(D1) (T, error)) ProviderFunc {
	return NewProviderFunc(
		func(c *Container) { Provide1TErr(c, fn) },
		ProviderMetadata{
			Label:        "ProviderErr1",
			Output:       TypedService[T](),
			Dependencies: ServiceRefs(TypedService[D1]()),
		},
	)
}

// Provider2 registers a typed singleton provider with two dependencies.
func Provider2[T, D1, D2 any](fn func(D1, D2) T) ProviderFunc {
	return NewProviderFunc(
		func(c *Container) { Provide2T(c, fn) },
		ProviderMetadata{
			Label:        "Provider2",
			Output:       TypedService[T](),
			Dependencies: ServiceRefs(TypedService[D1](), TypedService[D2]()),
		},
	)
}

// ProviderErr2 registers a typed singleton provider with two dependencies.
func ProviderErr2[T, D1, D2 any](fn func(D1, D2) (T, error)) ProviderFunc {
	return NewProviderFunc(
		func(c *Container) { Provide2TErr(c, fn) },
		ProviderMetadata{
			Label:        "ProviderErr2",
			Output:       TypedService[T](),
			Dependencies: ServiceRefs(TypedService[D1](), TypedService[D2]()),
		},
	)
}

// Provider3 registers a typed singleton provider with three dependencies.
func Provider3[T, D1, D2, D3 any](fn func(D1, D2, D3) T) ProviderFunc {
	return NewProviderFunc(
		func(c *Container) { Provide3T(c, fn) },
		ProviderMetadata{
			Label:        "Provider3",
			Output:       TypedService[T](),
			Dependencies: ServiceRefs(TypedService[D1](), TypedService[D2](), TypedService[D3]()),
		},
	)
}

// ProviderErr3 registers a typed singleton provider with three dependencies.
func ProviderErr3[T, D1, D2, D3 any](fn func(D1, D2, D3) (T, error)) ProviderFunc {
	return NewProviderFunc(
		func(c *Container) { Provide3TErr(c, fn) },
		ProviderMetadata{
			Label:        "ProviderErr3",
			Output:       TypedService[T](),
			Dependencies: ServiceRefs(TypedService[D1](), TypedService[D2](), TypedService[D3]()),
		},
	)
}

// Provider4 registers a typed singleton provider with four dependencies.
func Provider4[T, D1, D2, D3, D4 any](fn func(D1, D2, D3, D4) T) ProviderFunc {
	return NewProviderFunc(
		func(c *Container) { Provide4T(c, fn) },
		ProviderMetadata{
			Label:        "Provider4",
			Output:       TypedService[T](),
			Dependencies: ServiceRefs(TypedService[D1](), TypedService[D2](), TypedService[D3](), TypedService[D4]()),
		},
	)
}

// ProviderErr4 registers a typed singleton provider with four dependencies.
func ProviderErr4[T, D1, D2, D3, D4 any](fn func(D1, D2, D3, D4) (T, error)) ProviderFunc {
	return NewProviderFunc(
		func(c *Container) { Provide4TErr(c, fn) },
		ProviderMetadata{
			Label:        "ProviderErr4",
			Output:       TypedService[T](),
			Dependencies: ServiceRefs(TypedService[D1](), TypedService[D2](), TypedService[D3](), TypedService[D4]()),
		},
	)
}

// Provider5 registers a typed singleton provider with five dependencies.
func Provider5[T, D1, D2, D3, D4, D5 any](fn func(D1, D2, D3, D4, D5) T) ProviderFunc {
	return NewProviderFunc(
		func(c *Container) { Provide5T(c, fn) },
		ProviderMetadata{
			Label:        "Provider5",
			Output:       TypedService[T](),
			Dependencies: ServiceRefs(TypedService[D1](), TypedService[D2](), TypedService[D3](), TypedService[D4](), TypedService[D5]()),
		},
	)
}

// ProviderErr5 registers a typed singleton provider with five dependencies.
func ProviderErr5[T, D1, D2, D3, D4, D5 any](fn func(D1, D2, D3, D4, D5) (T, error)) ProviderFunc {
	return NewProviderFunc(
		func(c *Container) { Provide5TErr(c, fn) },
		ProviderMetadata{
			Label:        "ProviderErr5",
			Output:       TypedService[T](),
			Dependencies: ServiceRefs(TypedService[D1](), TypedService[D2](), TypedService[D3](), TypedService[D4](), TypedService[D5]()),
		},
	)
}

// Provider6 registers a typed singleton provider with six dependencies.
func Provider6[T, D1, D2, D3, D4, D5, D6 any](fn func(D1, D2, D3, D4, D5, D6) T) ProviderFunc {
	return NewProviderFunc(
		func(c *Container) { Provide6T(c, fn) },
		ProviderMetadata{
			Label:  "Provider6",
			Output: TypedService[T](),
			Dependencies: ServiceRefs(
				TypedService[D1](),
				TypedService[D2](),
				TypedService[D3](),
				TypedService[D4](),
				TypedService[D5](),
				TypedService[D6](),
			),
		},
	)
}

// ProviderErr6 registers a typed singleton provider with six dependencies.
func ProviderErr6[T, D1, D2, D3, D4, D5, D6 any](fn func(D1, D2, D3, D4, D5, D6) (T, error)) ProviderFunc {
	return NewProviderFunc(
		func(c *Container) { Provide6TErr(c, fn) },
		ProviderMetadata{
			Label:  "ProviderErr6",
			Output: TypedService[T](),
			Dependencies: ServiceRefs(
				TypedService[D1](),
				TypedService[D2](),
				TypedService[D3](),
				TypedService[D4](),
				TypedService[D5](),
				TypedService[D6](),
			),
		},
	)
}
