package advanced

import (
	"github.com/DaiYuANg/arcgo/dix"
	"github.com/samber/do/v2"
)

// DoSetup registers a raw setup against a do injector.
func DoSetup(fn func(do.Injector) error) dix.SetupFunc {
	return DoSetupWithMetadata(fn, dix.SetupMetadata{
		Label:         "DoSetup",
		GraphMutation: true,
	})
}

// DoSetupWithMetadata registers a raw setup against a do injector with metadata.
func DoSetupWithMetadata(fn func(do.Injector) error, meta dix.SetupMetadata) dix.SetupFunc {
	meta.Raw = true
	return dix.NewSetupFunc(func(c *dix.Container, _ dix.Lifecycle) error {
		return fn(c.Raw())
	}, meta)
}

// BindAlias binds one typed service to another interface or alias type.
func BindAlias[From, To any]() dix.SetupFunc {
	return newSetup("BindAlias", func(c *dix.Container) error {
		return do.As[From, To](c.Raw())
	}, dix.ServiceRefs(dix.TypedService[From]()), dix.ServiceRefs(dix.TypedService[To]()), nil)
}

// BindNamedAlias binds one named service to another named alias.
func BindNamedAlias[From, To any](sourceName, aliasName string) dix.SetupFunc {
	return newSetup("BindNamedAlias", func(c *dix.Container) error {
		return do.AsNamed[From, To](c.Raw(), sourceName, aliasName)
	}, dix.ServiceRefs(dix.NamedService(sourceName)), dix.ServiceRefs(dix.NamedService(aliasName)), nil)
}

// OverrideValue overrides a typed value registration.
func OverrideValue[T any](value T) dix.SetupFunc {
	return NamedOverrideValue(typedName[T](), value)
}

// NamedOverrideValue overrides a named value registration.
func NamedOverrideValue[T any](name string, value T) dix.SetupFunc {
	return newSetup("OverrideValue", func(c *dix.Container) error {
		do.OverrideNamedValue(c.Raw(), name, value)
		return nil
	}, nil, nil, dix.ServiceRefs(dix.NamedService(name)))
}

// Override0 overrides a typed provider with no dependencies.
func Override0[T any](fn func() T) dix.SetupFunc {
	return NamedOverride0(typedName[T](), fn)
}

// NamedOverride0 overrides a named provider with no dependencies.
func NamedOverride0[T any](name string, fn func() T) dix.SetupFunc {
	return newSetup("Override0", func(c *dix.Container) error {
		do.OverrideNamed(c.Raw(), name, func(do.Injector) (T, error) { return fn(), nil })
		return nil
	}, nil, nil, dix.ServiceRefs(dix.NamedService(name)))
}

// Override1 overrides a typed provider with one dependency.
func Override1[T, D1 any](fn func(D1) T) dix.SetupFunc {
	return NamedOverride1(typedName[T](), fn)
}

// NamedOverride1 overrides a named provider with one dependency.
func NamedOverride1[T, D1 any](name string, fn func(D1) T) dix.SetupFunc {
	return newSetup("Override1", func(c *dix.Container) error {
		do.OverrideNamed(c.Raw(), name, func(i do.Injector) (T, error) {
			d1, err := invokeTyped[D1](i)
			if err != nil {
				var zero T
				return zero, err
			}
			return fn(d1), nil
		})
		return nil
	}, dix.ServiceRefs(dix.TypedService[D1]()), nil, dix.ServiceRefs(dix.NamedService(name)))
}

// OverrideTransient0 overrides a typed transient provider with no dependencies.
func OverrideTransient0[T any](fn func() T) dix.SetupFunc {
	return NamedOverrideTransient0(typedName[T](), fn)
}

// NamedOverrideTransient0 overrides a named transient provider with no dependencies.
func NamedOverrideTransient0[T any](name string, fn func() T) dix.SetupFunc {
	return newSetup("OverrideTransient0", func(c *dix.Container) error {
		do.OverrideNamedTransient(c.Raw(), name, func(do.Injector) (T, error) { return fn(), nil })
		return nil
	}, nil, nil, dix.ServiceRefs(dix.NamedService(name)))
}

// OverrideTransient1 overrides a typed transient provider with one dependency.
func OverrideTransient1[T, D1 any](fn func(D1) T) dix.SetupFunc {
	return NamedOverrideTransient1(typedName[T](), fn)
}

// NamedOverrideTransient1 overrides a named transient provider with one dependency.
func NamedOverrideTransient1[T, D1 any](name string, fn func(D1) T) dix.SetupFunc {
	return newSetup("OverrideTransient1", func(c *dix.Container) error {
		do.OverrideNamedTransient(c.Raw(), name, func(i do.Injector) (T, error) {
			d1, err := invokeTyped[D1](i)
			if err != nil {
				var zero T
				return zero, err
			}
			return fn(d1), nil
		})
		return nil
	}, dix.ServiceRefs(dix.TypedService[D1]()), nil, dix.ServiceRefs(dix.NamedService(name)))
}
