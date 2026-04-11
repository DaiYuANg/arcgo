package dix

import (
	"log/slog"

	"github.com/DaiYuANg/arcgo/collectionx"
	collectionlist "github.com/DaiYuANg/arcgo/collectionx/list"
	collectionset "github.com/DaiYuANg/arcgo/collectionx/set"
	"github.com/samber/oops"
)

func validateTypedGraphReport(plan *buildPlan) ValidationReport {
	if plan == nil || plan.modules == nil {
		return ValidationReport{}
	}

	state := newValidationState()
	collectDeclaredOutputs(plan.modules, state)
	validateDeclaredDependencies(plan.modules, state)

	return ValidationReport{
		Errors:   collectionx.NewListWithCapacity(state.err.Len(), state.err.Values()...),
		Warnings: collectionx.NewListWithCapacity(state.warnings.Len(), state.warnings.Values()...),
	}
}

type validationState struct {
	known    *collectionset.Set[string]
	err      *collectionlist.List[error]
	warnings *collectionlist.List[ValidationWarning]
}

func newValidationState() *validationState {
	return &validationState{
		known: collectionset.NewSetWithCapacity[string](64,
			serviceNameOf[*slog.Logger](),
			serviceNameOf[AppMeta](),
			serviceNameOf[Profile](),
		),
		err:      collectionlist.NewListWithCapacity[error](4),
		warnings: collectionlist.NewListWithCapacity[ValidationWarning](2),
	}
}

func collectDeclaredOutputs(modules *collectionlist.List[*moduleSpec], state *validationState) {
	modules.Range(func(_ int, mod *moduleSpec) bool {
		if mod == nil {
			return true
		}
		collectProviderOutputs(mod, state)
		collectSetupOutputs(mod, state)
		return true
	})
}

func collectProviderOutputs(mod *moduleSpec, state *validationState) {
	mod.providers.Range(func(_ int, provider ProviderFunc) bool {
		meta := provider.meta
		if meta.Output.Name != "" {
			if state.known.Contains(meta.Output.Name) {
				state.err.Add(oops.In("dix").
					With("op", "validate_provider_output", "module", mod.name, "label", meta.Label, "service", meta.Output.Name).
					Errorf("duplicate provider output `%s` in module `%s` via %s", meta.Output.Name, mod.name, meta.Label))
				return true
			}
			state.known.Add(meta.Output.Name)
			return true
		}
		if meta.Raw {
			state.addWarning(
				ValidationWarningRawProviderUndeclaredOutput,
				mod.name,
				meta.Label,
				"raw provider has no declared output; validation cannot model services it registers",
			)
		}
		return true
	})
}

func collectSetupOutputs(mod *moduleSpec, state *validationState) {
	mod.setups.Range(func(_ int, setup SetupFunc) bool {
		meta := setup.meta
		meta.Provides.Range(func(_ int, provide ServiceRef) bool {
			if state.known.Contains(provide.Name) {
				state.err.Add(oops.In("dix").
					With("op", "validate_setup_output", "module", mod.name, "label", meta.Label, "service", provide.Name).
					Errorf("duplicate setup output `%s` in module `%s` via %s", provide.Name, mod.name, meta.Label))
				return true
			}
			state.known.Add(provide.Name)
			return true
		})
		if meta.Raw && meta.Provides.Len() == 0 && meta.Overrides.Len() == 0 && meta.GraphMutation {
			state.addWarning(
				ValidationWarningRawSetupUndeclaredGraph,
				mod.name,
				meta.Label,
				"raw setup mutates the graph without declared provides/overrides; validation cannot model its graph effects",
			)
		}
		return true
	})
}

func validateDeclaredDependencies(modules *collectionlist.List[*moduleSpec], state *validationState) {
	modules.Range(func(_ int, mod *moduleSpec) bool {
		if mod == nil {
			return true
		}
		validateProviderDependencies(mod, state)
		validateSetupDependencies(mod, state)
		validateInvokeDependencies(mod, state)
		validateHookDependencies(mod, state)
		return true
	})
}

func validateProviderDependencies(mod *moduleSpec, state *validationState) {
	mod.providers.Range(func(_ int, provider ProviderFunc) bool {
		meta := provider.meta
		if meta.Raw && meta.Dependencies.Len() == 0 {
			state.addWarning(
				ValidationWarningRawProviderUndeclaredDeps,
				mod.name,
				meta.Label,
				"raw provider has no declared dependencies; validation cannot verify what it resolves at registration time",
			)
		}
		state.validateDeps(mod.name, "provider", meta.Label, meta.Dependencies)
		return true
	})
}

func validateSetupDependencies(mod *moduleSpec, state *validationState) {
	mod.setups.Range(func(_ int, setup SetupFunc) bool {
		meta := setup.meta
		meta.Overrides.Range(func(_ int, override ServiceRef) bool {
			if !state.known.Contains(override.Name) {
				state.err.Add(oops.In("dix").
					With("op", "validate_setup_override", "module", mod.name, "label", meta.Label, "service", override.Name).
					Errorf("override target `%s` not found in module `%s` via %s", override.Name, mod.name, meta.Label))
			}
			return true
		})
		state.validateDeps(mod.name, "setup", meta.Label, meta.Dependencies)
		return true
	})
}

func validateInvokeDependencies(mod *moduleSpec, state *validationState) {
	mod.invokes.Range(func(_ int, invoke InvokeFunc) bool {
		if invoke.meta.Raw && invoke.meta.Dependencies.Len() == 0 {
			state.addWarning(
				ValidationWarningRawInvokeUndeclaredDeps,
				mod.name,
				invoke.meta.Label,
				"raw invoke has no declared dependencies; validation cannot verify what it resolves",
			)
			return true
		}
		state.validateDeps(mod.name, "invoke", invoke.meta.Label, invoke.meta.Dependencies)
		return true
	})
}

func validateHookDependencies(mod *moduleSpec, state *validationState) {
	mod.hooks.Range(func(_ int, hook HookFunc) bool {
		if hook.meta.Raw && hook.meta.Dependencies.Len() == 0 {
			state.addWarning(
				ValidationWarningRawHookUndeclaredDeps,
				mod.name,
				hook.meta.Label,
				"raw hook has no declared dependencies; validation cannot verify what it resolves during lifecycle execution",
			)
			return true
		}
		state.validateDeps(mod.name, string(hook.meta.Kind)+" hook", hook.meta.Label, hook.meta.Dependencies)
		return true
	})
}

func (s *validationState) addWarning(kind ValidationWarningKind, moduleName, label, details string) {
	s.warnings.Add(ValidationWarning{
		Kind:    kind,
		Module:  moduleName,
		Label:   label,
		Details: details,
	})
}

func (s *validationState) validateDeps(moduleName, kind, label string, deps collectionx.List[ServiceRef]) {
	validateDependencies(s.err, s.known, moduleName, kind, label, deps)
}

func validateDependencies(
	err *collectionlist.List[error],
	known *collectionset.Set[string],
	moduleName string,
	kind string,
	label string,
	deps collectionx.List[ServiceRef],
) {
	deps.Range(func(_ int, dep ServiceRef) bool {
		if !known.Contains(dep.Name) {
			err.Add(oops.In("dix").
				With("op", "validate_dependency", "module", moduleName, "label", label, "dependency", dep.Name, "kind", kind).
				Errorf("missing dependency `%s` for %s %s in module `%s`", dep.Name, kind, label, moduleName))
		}
		return true
	})
}
