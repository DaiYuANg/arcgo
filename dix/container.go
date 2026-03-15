package dix

import (
	"context"

	do "github.com/samber/do/v2"
)

type Container interface {
	Register(def Definition) error
	Resolve(target any) error
	Shutdown(ctx context.Context) error
}

type DefinitionKind string

const (
	DefinitionValue    DefinitionKind = "value"
	DefinitionProvider DefinitionKind = "provider"
)

type Definition struct {
	Name string
	Kind DefinitionKind

	// 二选一
	Value    any
	Provider any

	// 可选元信息
	ModuleName string
	Lazy       bool
	Transient  bool
}

type BuildPlan struct {
	Profile Profile

	Modules         []Module
	EnabledModules  []Module
	DisabledModules []Module

	Definitions []Definition

	Invokes     []Callable
	StartHooks  []Callable
	StopHooks   []Callable
	HealthHooks []Callable
}

type doContainer struct {
	injector do.Injector
}
