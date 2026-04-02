package dix

import "github.com/DaiYuANg/arcgo/collectionx"
import "github.com/DaiYuANg/arcgo/pkg/option"

// ModuleOption configures a Module during construction.
type ModuleOption func(*moduleSpec)

// NewModule creates an immutable module specification.
func NewModule(name string, opts ...ModuleOption) Module {
	spec := &moduleSpec{name: name}
	option.Apply(spec, opts...)
	return Module{spec: spec}
}

// Name returns the module name.
func (m Module) Name() string {
	if m.spec == nil {
		return ""
	}
	return m.spec.name
}

// Description returns the module description.
func (m Module) Description() string {
	if m.spec == nil {
		return ""
	}
	return m.spec.description
}

// Tags returns the module tags.
func (m Module) Tags() collectionx.OrderedSet[string] {
	if m.spec == nil {
		return collectionx.NewOrderedSet[string]()
	}
	return m.spec.tags.Clone()
}

// Imports returns the imported modules.
func (m Module) Imports() collectionx.List[Module] {
	if m.spec == nil {
		return collectionx.NewList[Module]()
	}
	return m.spec.imports.Clone()
}
