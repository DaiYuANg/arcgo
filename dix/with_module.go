package dix

// WithModuleProviders appends provider registrations to a module.
func WithModuleProviders(providers ...ProviderFunc) ModuleOption {
	return func(spec *moduleSpec) { spec.providers.Add(providers...) }
}

// WithModuleProvider appends a single provider registration to a module.
func WithModuleProvider(provider ProviderFunc) ModuleOption {
	return WithModuleProviders(provider)
}

// WithModuleSetups appends setup registrations to a module.
func WithModuleSetups(setups ...SetupFunc) ModuleOption {
	return func(spec *moduleSpec) { spec.setups.Add(setups...) }
}

// WithModuleInvokes appends invoke registrations to a module.
func WithModuleInvokes(invokes ...InvokeFunc) ModuleOption {
	return func(spec *moduleSpec) { spec.invokes.Add(invokes...) }
}

// WithModuleInvoke appends a single invoke registration to a module.
func WithModuleInvoke(invoke InvokeFunc) ModuleOption {
	return WithModuleInvokes(invoke)
}

// WithModuleHooks appends lifecycle hook registrations to a module.
func WithModuleHooks(hooks ...HookFunc) ModuleOption {
	return func(spec *moduleSpec) { spec.hooks.Add(hooks...) }
}

// WithModuleHook appends a single lifecycle hook registration to a module.
func WithModuleHook(hook HookFunc) ModuleOption {
	return WithModuleHooks(hook)
}

// WithModuleImports appends imported modules to a module.
func WithModuleImports(modules ...Module) ModuleOption {
	return func(spec *moduleSpec) { spec.imports.Add(modules...) }
}

// WithModuleImport appends a single imported module to a module.
func WithModuleImport(module Module) ModuleOption {
	return WithModuleImports(module)
}

// WithModuleProfiles restricts a module to the listed profiles.
func WithModuleProfiles(profiles ...Profile) ModuleOption {
	return func(spec *moduleSpec) { spec.profiles.Add(profiles...) }
}

// WithModuleProfile restricts a module to a single profile.
func WithModuleProfile(profile Profile) ModuleOption {
	return WithModuleProfiles(profile)
}

// WithModuleExcludeProfiles excludes a module from the listed profiles.
func WithModuleExcludeProfiles(profiles ...Profile) ModuleOption {
	return func(spec *moduleSpec) { spec.excludeProfiles.Add(profiles...) }
}

// WithModuleExcludeProfile excludes a module from a single profile.
func WithModuleExcludeProfile(profile Profile) ModuleOption {
	return WithModuleExcludeProfiles(profile)
}

// WithModuleDescription sets the module description.
func WithModuleDescription(desc string) ModuleOption {
	return func(spec *moduleSpec) { spec.description = desc }
}

// WithModuleTags appends tags to a module.
func WithModuleTags(tags ...string) ModuleOption {
	return func(spec *moduleSpec) { spec.tags.Add(tags...) }
}

// WithModuleSetup appends a typed setup callback to a module.
func WithModuleSetup(fn func(*Container, Lifecycle) error) ModuleOption {
	return WithModuleSetups(Setup(fn))
}

// WithModuleDisabled sets whether the module is disabled.
func WithModuleDisabled(disabled bool) ModuleOption {
	return func(spec *moduleSpec) { spec.disabled = disabled }
}
