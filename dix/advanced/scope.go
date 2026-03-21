package advanced

import (
	"github.com/DaiYuANg/arcgo/dix"
	do "github.com/samber/do/v2"
	"github.com/samber/lo"
)

type ScopePackage func(do.Injector)

func Scope(rt *dix.Runtime, name string, packages ...ScopePackage) *do.Scope {
	if rt == nil {
		return nil
	}

	valid := lo.Filter(packages, func(pkg ScopePackage, _ int) bool { return pkg != nil })
	switch len(valid) {
	case 0:
		return rt.Raw().Scope(name)
	case 1:
		current := valid[0]
		return rt.Raw().Scope(name, func(injector do.Injector) {
			current(injector)
		})
	default:
		wrapped := lo.Map(valid, func(pkg ScopePackage, _ int) func(do.Injector) {
			current := pkg
			return func(injector do.Injector) {
				current(injector)
			}
		})
		return rt.Raw().Scope(name, wrapped...)
	}
}

func ProvideScopedValue[T any](injector do.Injector, value T) {
	do.ProvideNamedValue(injector, typedName[T](), value)
}

func ProvideScopedNamedValue[T any](injector do.Injector, name string, value T) {
	do.ProvideNamedValue(injector, name, value)
}

func ProvideScoped0[T any](injector do.Injector, fn func() T) {
	do.ProvideNamed(injector, typedName[T](), func(do.Injector) (T, error) {
		return fn(), nil
	})
}

func ProvideScopedNamed0[T any](injector do.Injector, name string, fn func() T) {
	do.ProvideNamed(injector, name, func(do.Injector) (T, error) {
		return fn(), nil
	})
}

func ProvideScoped1[T, D1 any](injector do.Injector, fn func(D1) T) {
	do.ProvideNamed(injector, typedName[T](), func(i do.Injector) (T, error) {
		d1, err := invokeTyped[D1](i)
		if err != nil {
			var zero T
			return zero, err
		}
		return fn(d1), nil
	})
}

func ProvideScopedNamed1[T, D1 any](injector do.Injector, name string, fn func(D1) T) {
	do.ProvideNamed(injector, name, func(i do.Injector) (T, error) {
		d1, err := invokeTyped[D1](i)
		if err != nil {
			var zero T
			return zero, err
		}
		return fn(d1), nil
	})
}

func ProvideScoped2[T, D1, D2 any](injector do.Injector, fn func(D1, D2) T) {
	do.ProvideNamed(injector, typedName[T](), func(i do.Injector) (T, error) {
		d1, err := invokeTyped[D1](i)
		if err != nil {
			var zero T
			return zero, err
		}
		d2, err := invokeTyped[D2](i)
		if err != nil {
			var zero T
			return zero, err
		}
		return fn(d1, d2), nil
	})
}

func ProvideScopedNamed2[T, D1, D2 any](injector do.Injector, name string, fn func(D1, D2) T) {
	do.ProvideNamed(injector, name, func(i do.Injector) (T, error) {
		d1, err := invokeTyped[D1](i)
		if err != nil {
			var zero T
			return zero, err
		}
		d2, err := invokeTyped[D2](i)
		if err != nil {
			var zero T
			return zero, err
		}
		return fn(d1, d2), nil
	})
}

func ProvideScoped3[T, D1, D2, D3 any](injector do.Injector, fn func(D1, D2, D3) T) {
	do.ProvideNamed(injector, typedName[T](), func(i do.Injector) (T, error) {
		d1, err := invokeTyped[D1](i)
		if err != nil {
			var zero T
			return zero, err
		}
		d2, err := invokeTyped[D2](i)
		if err != nil {
			var zero T
			return zero, err
		}
		d3, err := invokeTyped[D3](i)
		if err != nil {
			var zero T
			return zero, err
		}
		return fn(d1, d2, d3), nil
	})
}

func ProvideScopedNamed3[T, D1, D2, D3 any](injector do.Injector, name string, fn func(D1, D2, D3) T) {
	do.ProvideNamed(injector, name, func(i do.Injector) (T, error) {
		d1, err := invokeTyped[D1](i)
		if err != nil {
			var zero T
			return zero, err
		}
		d2, err := invokeTyped[D2](i)
		if err != nil {
			var zero T
			return zero, err
		}
		d3, err := invokeTyped[D3](i)
		if err != nil {
			var zero T
			return zero, err
		}
		return fn(d1, d2, d3), nil
	})
}

func ResolveScopedAs[T any](injector do.Injector) (T, error) {
	return ResolveInjectorAs[T](injector)
}

func ResolveScopedNamedAs[T any](injector do.Injector, name string) (T, error) {
	return do.InvokeNamed[T](injector, name)
}
