package dix

import (
	"errors"
	"fmt"
	"log/slog"

	collectionlist "github.com/DaiYuANg/arcgo/collectionx/list"
	collectionset "github.com/DaiYuANg/arcgo/collectionx/set"
)

func validateTypedGraph(plan *buildPlan) error {
	if plan == nil || plan.modules == nil {
		return nil
	}

	known := collectionset.NewSetWithCapacity[string](64,
		serviceNameOf[*slog.Logger](),
		serviceNameOf[AppMeta](),
		serviceNameOf[Profile](),
	)
	errs := collectionlist.NewListWithCapacity[error](4)
	graphEscapes := false

	validateDeps := func(modName, kind, label string, deps []ServiceRef) {
		if graphEscapes {
			return
		}
		validateDependencies(errs, known, modName, kind, label, deps)
	}

	plan.modules.Range(func(_ int, mod *moduleSpec) bool {
		if mod == nil {
			return true
		}
		modName := mod.name

		mod.providers.Range(func(_ int, provider ProviderFunc) bool {
			meta := provider.meta
			if meta.Raw {
				graphEscapes = true
				return true
			}
			if meta.Output.Name != "" {
				if known.Contains(meta.Output.Name) {
					errs.Add(fmt.Errorf("duplicate provider output `%s` in module `%s` via %s", meta.Output.Name, modName, meta.Label))
					return true
				}
				known.Add(meta.Output.Name)
			}
			validateDeps(modName, "provider", meta.Label, meta.Dependencies)
			return true
		})

		mod.setups.Range(func(_ int, setup SetupFunc) bool {
			meta := setup.meta
			if meta.Raw || (meta.GraphMutation && len(meta.Provides) == 0 && len(meta.Overrides) == 0) {
				graphEscapes = true
			}
			for _, provide := range meta.Provides {
				if known.Contains(provide.Name) {
					errs.Add(fmt.Errorf("duplicate setup output `%s` in module `%s` via %s", provide.Name, modName, meta.Label))
					continue
				}
				known.Add(provide.Name)
			}
			for _, override := range meta.Overrides {
				if !known.Contains(override.Name) {
					errs.Add(fmt.Errorf("override target `%s` not found in module `%s` via %s", override.Name, modName, meta.Label))
					continue
				}
				known.Add(override.Name)
			}
			validateDeps(modName, "setup", meta.Label, meta.Dependencies)
			return true
		})

		mod.invokes.Range(func(_ int, invoke InvokeFunc) bool {
			if invoke.meta.Raw {
				graphEscapes = true
			}
			validateDeps(modName, "invoke", invoke.meta.Label, invoke.meta.Dependencies)
			return true
		})

		mod.hooks.Range(func(_ int, hook HookFunc) bool {
			if hook.meta.Raw {
				graphEscapes = true
			}
			validateDeps(modName, string(hook.meta.Kind)+" hook", hook.meta.Label, hook.meta.Dependencies)
			return true
		})

		return true
	})

	if errs.IsEmpty() {
		return nil
	}

	return errors.Join(errs.Values()...)
}

func validateDependencies(
	errs *collectionlist.List[error],
	known *collectionset.Set[string],
	moduleName string,
	kind string,
	label string,
	deps []ServiceRef,
) {
	for _, dep := range deps {
		if !known.Contains(dep.Name) {
			errs.Add(fmt.Errorf("missing dependency `%s` for %s %s in module `%s`", dep.Name, kind, label, moduleName))
		}
	}
}
