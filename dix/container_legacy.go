package dix

import (
	"errors"
	"fmt"

	"github.com/samber/do/v2"
)

// Definition describes a backward-compatible container registration.
type Definition struct {
	Name       string
	Kind       DefinitionKind
	Value      any
	Provider   any
	ModuleName string
	Lazy       bool
	Transient  bool
}

// DefinitionKind describes the kind of backward-compatible registration.
type DefinitionKind string

const (
	// DefinitionValue registers an already-constructed value.
	DefinitionValue DefinitionKind = "value"
	// DefinitionProvider registers a provider function.
	DefinitionProvider DefinitionKind = "provider"
)

// Register registers a backward-compatible definition.
func (c *Container) Register(def Definition) error {
	switch def.Kind {
	case DefinitionValue:
		if def.Name != "" {
			do.ProvideNamedValue(c.injector, def.Name, def.Value)
		} else {
			do.ProvideValue(c.injector, def.Value)
		}
		return nil
	case DefinitionProvider:
		return errors.New("provider definition registration is not implemented; use typed ProviderN helpers instead")
	default:
		return fmt.Errorf("unknown definition kind: %v", def.Kind)
	}
}

// Resolve keeps backward compatibility for legacy resolve(target) calls.
func (c *Container) Resolve(any) error {
	return errors.New("resolve(target) is not supported; use ResolveAs[T]() for type-safe resolution")
}
