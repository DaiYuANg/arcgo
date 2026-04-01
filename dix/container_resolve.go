package dix

import (
	"fmt"

	"github.com/samber/mo"
)

// ResolveAs resolves a typed value from the container.
func ResolveAs[T any](c *Container) (T, error) { return resolveInjectorAs[T](c.injector) }

// ResolveOptionalAs resolves an optional typed value from the container.
func ResolveOptionalAs[T any](c *Container) (value T, ok bool) {
	option := ResolveOptionAs[T](c)
	return option.Get()
}

// ResolveOptionAs resolves an optional dependency as mo.Option.
func ResolveOptionAs[T any](c *Container) mo.Option[T] {
	value, err := ResolveAs[T](c)
	if err == nil {
		return mo.Some(value)
	}
	return mo.None[T]()
}

// ResolveOrElse resolves a typed value or returns the provided fallback.
func ResolveOrElse[T any](c *Container, fallback T) T {
	return ResolveOptionAs[T](c).OrElse(fallback)
}

// MustResolveAs resolves a typed value and panics on failure.
func MustResolveAs[T any](c *Container) T {
	result, err := ResolveAs[T](c)
	if err != nil {
		panic(fmt.Sprintf("failed to resolve dependency: %v", err))
	}
	return result
}
