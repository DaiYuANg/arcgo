package dix

import (
	"context"
	"log/slog"

	collectionlist "github.com/DaiYuANg/arcgo/collectionx/list"
	"github.com/samber/do/v2"
)

// Container wraps samber/do.Injector.
// Most code should stay on the typed dix helpers.
// Raw() exists as an explicit escape hatch for advanced integrations.
type Container struct {
	injector     do.Injector
	healthChecks collectionlist.List[healthCheckEntry]
	logger       *slog.Logger
}

func newContainer() *Container {
	return &Container{
		injector: do.New(),
		logger:   slog.Default(),
	}
}

// Raw returns the underlying do injector for advanced integrations.
func (c *Container) Raw() do.Injector { return c.injector }

// Injector returns the underlying do injector.
//
// Deprecated: prefer Raw() to make advanced usage explicit at call sites.
func (c *Container) Injector() do.Injector { return c.injector }

// Shutdown shuts down all registered container services.
func (c *Container) Shutdown(ctx context.Context) error {
	report := c.ShutdownReport(ctx)
	if report == nil || len(report.Errors) == 0 {
		return nil
	}
	return report
}

// ShutdownReport shuts down the container and returns the do shutdown report.
func (c *Container) ShutdownReport(ctx context.Context) *do.ShutdownReport {
	if c == nil || c.injector == nil {
		return nil
	}
	if c.logger != nil && c.logger.Enabled(ctx, slog.LevelDebug) {
		c.logger.Debug("shutting down container")
	}
	report := c.injector.ShutdownWithContext(ctx)
	if c.logger != nil && c.logger.Enabled(ctx, slog.LevelDebug) {
		errorsCount := 0
		if report != nil {
			errorsCount = len(report.Errors)
		}
		c.logger.Debug("container shutdown completed", "errors", errorsCount)
	}
	return report
}

func resolveInjectorAs[T any](injector do.Injector) (T, error) {
	return do.InvokeNamed[T](injector, serviceNameOf[T]())
}

// ProvideT registers a typed singleton provider with no dependencies.
func ProvideT[T any](c *Container, fn func() T) {
	ProvideTErr(c, func() (T, error) { return fn(), nil })
}

// ProvideTErr registers a typed singleton provider with no dependencies.
func ProvideTErr[T any](c *Container, fn func() (T, error)) {
	do.ProvideNamed(c.injector, serviceNameOf[T](), func(_ do.Injector) (T, error) { return fn() })
}

// Provide1T registers a typed singleton provider with one dependency.
func Provide1T[T, D1 any](c *Container, fn func(D1) T) {
	Provide1TErr(c, func(d1 D1) (T, error) { return fn(d1), nil })
}

// Provide1TErr registers a typed singleton provider with one dependency.
func Provide1TErr[T, D1 any](c *Container, fn func(D1) (T, error)) {
	do.ProvideNamed(c.injector, serviceNameOf[T](), func(i do.Injector) (T, error) {
		d1, err := resolveInjectorAs[D1](i)
		if err != nil {
			var zero T
			return zero, err
		}
		return fn(d1)
	})
}

// Provide2T registers a typed singleton provider with two dependencies.
func Provide2T[T, D1, D2 any](c *Container, fn func(D1, D2) T) {
	Provide2TErr(c, func(d1 D1, d2 D2) (T, error) { return fn(d1, d2), nil })
}

// Provide2TErr registers a typed singleton provider with two dependencies.
func Provide2TErr[T, D1, D2 any](c *Container, fn func(D1, D2) (T, error)) {
	do.ProvideNamed(c.injector, serviceNameOf[T](), func(i do.Injector) (T, error) {
		d1, err := resolveInjectorAs[D1](i)
		if err != nil {
			var zero T
			return zero, err
		}
		d2, err := resolveInjectorAs[D2](i)
		if err != nil {
			var zero T
			return zero, err
		}
		return fn(d1, d2)
	})
}

// Provide3T registers a typed singleton provider with three dependencies.
func Provide3T[T, D1, D2, D3 any](c *Container, fn func(D1, D2, D3) T) {
	Provide3TErr(c, func(d1 D1, d2 D2, d3 D3) (T, error) { return fn(d1, d2, d3), nil })
}

// Provide3TErr registers a typed singleton provider with three dependencies.
func Provide3TErr[T, D1, D2, D3 any](c *Container, fn func(D1, D2, D3) (T, error)) {
	do.ProvideNamed(c.injector, serviceNameOf[T](), func(i do.Injector) (T, error) {
		d1, err := resolveInjectorAs[D1](i)
		if err != nil {
			var zero T
			return zero, err
		}
		d2, err := resolveInjectorAs[D2](i)
		if err != nil {
			var zero T
			return zero, err
		}
		d3, err := resolveInjectorAs[D3](i)
		if err != nil {
			var zero T
			return zero, err
		}
		return fn(d1, d2, d3)
	})
}

// Provide4T registers a typed singleton provider with four dependencies.
func Provide4T[T, D1, D2, D3, D4 any](c *Container, fn func(D1, D2, D3, D4) T) {
	Provide4TErr(c, func(d1 D1, d2 D2, d3 D3, d4 D4) (T, error) { return fn(d1, d2, d3, d4), nil })
}

// Provide4TErr registers a typed singleton provider with four dependencies.
func Provide4TErr[T, D1, D2, D3, D4 any](c *Container, fn func(D1, D2, D3, D4) (T, error)) {
	do.ProvideNamed(c.injector, serviceNameOf[T](), func(i do.Injector) (T, error) {
		d1, err := resolveInjectorAs[D1](i)
		if err != nil {
			var zero T
			return zero, err
		}
		d2, err := resolveInjectorAs[D2](i)
		if err != nil {
			var zero T
			return zero, err
		}
		d3, err := resolveInjectorAs[D3](i)
		if err != nil {
			var zero T
			return zero, err
		}
		d4, err := resolveInjectorAs[D4](i)
		if err != nil {
			var zero T
			return zero, err
		}
		return fn(d1, d2, d3, d4)
	})
}

// Provide5T registers a typed singleton provider with five dependencies.
func Provide5T[T, D1, D2, D3, D4, D5 any](c *Container, fn func(D1, D2, D3, D4, D5) T) {
	Provide5TErr(c, func(d1 D1, d2 D2, d3 D3, d4 D4, d5 D5) (T, error) {
		return fn(d1, d2, d3, d4, d5), nil
	})
}

// Provide5TErr registers a typed singleton provider with five dependencies.
func Provide5TErr[T, D1, D2, D3, D4, D5 any](c *Container, fn func(D1, D2, D3, D4, D5) (T, error)) {
	do.ProvideNamed(c.injector, serviceNameOf[T](), func(i do.Injector) (T, error) {
		d1, err := resolveInjectorAs[D1](i)
		if err != nil {
			var zero T
			return zero, err
		}
		d2, err := resolveInjectorAs[D2](i)
		if err != nil {
			var zero T
			return zero, err
		}
		d3, err := resolveInjectorAs[D3](i)
		if err != nil {
			var zero T
			return zero, err
		}
		d4, err := resolveInjectorAs[D4](i)
		if err != nil {
			var zero T
			return zero, err
		}
		d5, err := resolveInjectorAs[D5](i)
		if err != nil {
			var zero T
			return zero, err
		}
		return fn(d1, d2, d3, d4, d5)
	})
}

// Provide6T registers a typed singleton provider with six dependencies.
func Provide6T[T, D1, D2, D3, D4, D5, D6 any](c *Container, fn func(D1, D2, D3, D4, D5, D6) T) {
	Provide6TErr(c, func(d1 D1, d2 D2, d3 D3, d4 D4, d5 D5, d6 D6) (T, error) {
		return fn(d1, d2, d3, d4, d5, d6), nil
	})
}

// Provide6TErr registers a typed singleton provider with six dependencies.
func Provide6TErr[T, D1, D2, D3, D4, D5, D6 any](c *Container, fn func(D1, D2, D3, D4, D5, D6) (T, error)) {
	do.ProvideNamed(c.injector, serviceNameOf[T](), func(i do.Injector) (T, error) {
		d1, d2, d3, d4, d5, d6, err := resolveProvide6Dependencies[D1, D2, D3, D4, D5, D6](i)
		if err != nil {
			var zero T
			return zero, err
		}
		return fn(d1, d2, d3, d4, d5, d6)
	})
}

//nolint:gocritic // Fixed-arity generic providers need typed multi-result dependency resolution.
func resolveProvide6Dependencies[D1, D2, D3, D4, D5, D6 any](injector do.Injector) (D1, D2, D3, D4, D5, D6, error) {
	d1, err := resolveInjectorAs[D1](injector)
	if err != nil {
		var zeroD1 D1
		var zeroD2 D2
		var zeroD3 D3
		var zeroD4 D4
		var zeroD5 D5
		var zeroD6 D6
		return zeroD1, zeroD2, zeroD3, zeroD4, zeroD5, zeroD6, err
	}
	d2, err := resolveInjectorAs[D2](injector)
	if err != nil {
		var zeroD2 D2
		var zeroD3 D3
		var zeroD4 D4
		var zeroD5 D5
		var zeroD6 D6
		return d1, zeroD2, zeroD3, zeroD4, zeroD5, zeroD6, err
	}
	d3, err := resolveInjectorAs[D3](injector)
	if err != nil {
		var zeroD3 D3
		var zeroD4 D4
		var zeroD5 D5
		var zeroD6 D6
		return d1, d2, zeroD3, zeroD4, zeroD5, zeroD6, err
	}
	d4, err := resolveInjectorAs[D4](injector)
	if err != nil {
		var zeroD4 D4
		var zeroD5 D5
		var zeroD6 D6
		return d1, d2, d3, zeroD4, zeroD5, zeroD6, err
	}
	d5, err := resolveInjectorAs[D5](injector)
	if err != nil {
		var zeroD5 D5
		var zeroD6 D6
		return d1, d2, d3, d4, zeroD5, zeroD6, err
	}
	d6, err := resolveInjectorAs[D6](injector)
	if err != nil {
		var zeroD6 D6
		return d1, d2, d3, d4, d5, zeroD6, err
	}
	return d1, d2, d3, d4, d5, d6, nil
}

// ProvideValueT registers a typed singleton value.
func ProvideValueT[T any](c *Container, value T) {
	do.ProvideNamedValue(c.injector, serviceNameOf[T](), value)
}
