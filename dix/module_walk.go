package dix

import (
	"fmt"
	"strings"

	collectionlist "github.com/DaiYuANg/arcgo/collectionx/list"
	collectionset "github.com/DaiYuANg/arcgo/collectionx/set"
)

type moduleVisitAction uint8

const (
	moduleVisitContinue moduleVisitAction = iota
	moduleVisitSkipChildren
	moduleVisitStop
)

type moduleVisitContext struct {
	Profile Profile
	Path    collectionlist.List[string]
	Depth   int
}

type moduleVisitor interface {
	Enter(ctx moduleVisitContext, spec *moduleSpec) (moduleVisitAction, error)
	Leave(ctx moduleVisitContext, spec *moduleSpec) error
}

type moduleVisitorFuncs struct {
	enter func(ctx moduleVisitContext, spec *moduleSpec) (moduleVisitAction, error)
	leave func(ctx moduleVisitContext, spec *moduleSpec) error
}

func (v moduleVisitorFuncs) Enter(ctx moduleVisitContext, spec *moduleSpec) (moduleVisitAction, error) {
	if v.enter == nil {
		return moduleVisitContinue, nil
	}
	return v.enter(ctx, spec)
}

func (v moduleVisitorFuncs) Leave(ctx moduleVisitContext, spec *moduleSpec) error {
	if v.leave == nil {
		return nil
	}
	return v.leave(ctx, spec)
}

// flattenModules walks active modules in dependency order and returns leaf-first results.
func flattenModules(modules []Module, profile Profile) (*collectionlist.List[*moduleSpec], error) {
	return flattenModuleList(collectionlist.NewListWithCapacity[Module](len(modules), modules...), profile)
}

func flattenModuleList(modules *collectionlist.List[Module], profile Profile) (*collectionlist.List[*moduleSpec], error) {
	capacity := 8
	if modules != nil && modules.Len() > 0 {
		if c := modules.Len() * 2; c > capacity {
			capacity = c
		}
	}
	result := collectionlist.NewListWithCapacity[*moduleSpec](capacity)

	err := walkModuleList(modules, profile, moduleVisitorFuncs{
		leave: func(_ moduleVisitContext, spec *moduleSpec) error {
			result.Add(spec)
			return nil
		},
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func walkModules(modules []Module, profile Profile, visitor moduleVisitor) error {
	return walkModuleList(collectionlist.NewListWithCapacity[Module](len(modules), modules...), profile, visitor)
}

func walkModuleList(modules *collectionlist.List[Module], profile Profile, visitor moduleVisitor) error {
	if modules == nil {
		return nil
	}

	visited := collectionset.NewSetWithCapacity[string](16)
	visiting := collectionset.NewSetWithCapacity[string](8)
	stopped := false
	path := collectionlist.NewListWithCapacity[string](8)

	var walk func(spec *moduleSpec) error
	walk = func(spec *moduleSpec) error {
		if stopped || spec == nil || spec.disabled || !isActiveForProfile(spec, profile) {
			return nil
		}

		key := moduleKey(spec)
		if visited.Contains(key) {
			return nil
		}
		if visiting.Contains(key) {
			return fmt.Errorf("module import cycle detected: %s -> %s", formatModulePath(*path), key)
		}

		path.Add(key)
		ctx := moduleVisitContext{
			Profile: profile,
			Path:    *path,
			Depth:   path.Len() - 1,
		}

		visiting.Add(key)
		action, err := visitor.Enter(ctx, spec)
		if err != nil {
			visiting.Remove(key)
			_, _ = path.RemoveAt(path.Len() - 1)
			return err
		}

		switch action {
		case moduleVisitStop:
			stopped = true
		case moduleVisitSkipChildren:
			// no-op
		default:
			var childErr error
			spec.imports.Range(func(_ int, imported Module) bool {
				childErr = walk(imported.spec)
				return childErr == nil && !stopped
			})
			if childErr != nil {
				visiting.Remove(key)
				_, _ = path.RemoveAt(path.Len() - 1)
				return childErr
			}
		}

		visiting.Remove(key)
		visited.Add(key)
		leaveErr := visitor.Leave(ctx, spec)
		_, _ = path.RemoveAt(path.Len() - 1)
		return leaveErr
	}

	var walkErr error
	modules.Range(func(_ int, mod Module) bool {
		walkErr = walk(mod.spec)
		return walkErr == nil && !stopped
	})

	return walkErr
}

func moduleKey(spec *moduleSpec) string {
	if spec == nil {
		return "<nil>"
	}
	if spec.name != "" {
		return spec.name
	}
	return fmt.Sprintf("<anonymous:%p>", spec)
}

func formatModulePath(path collectionlist.List[string]) string {
	if path.IsEmpty() {
		return "<root>"
	}
	return strings.Join(path.Values(), " -> ")
}

func isActiveForProfile(spec *moduleSpec, profile Profile) bool {
	if spec == nil || spec.disabled {
		return false
	}
	if spec.excludeProfiles.Contains(profile) {
		return false
	}
	if spec.profiles.IsEmpty() {
		return true
	}
	return spec.profiles.Contains(profile)
}
