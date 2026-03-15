package dix

import "sync/atomic"

type Profile string

const (
	ProfileDefault Profile = "default"
	ProfileDev     Profile = "dev"
	ProfileTest    Profile = "test"
	ProfileProd    Profile = "prod"
)

type AppMeta struct {
	Name        string
	Version     string
	Description string
}

type AppState int32

const (
	AppStateCreated AppState = iota
	AppStateBuilt
	AppStateStarted
	AppStateStopped
)

type Phase string

const (
	PhaseInvoke Phase = "invoke"
	PhaseStart  Phase = "start"
	PhaseStop   Phase = "stop"
	PhaseHealth Phase = "health"
)

type Callable struct {
	Name       string
	ModuleName string
	Phase      Phase
	Fn         any
}

type Module struct {
	Name        string
	Description string

	Profiles        []Profile
	ExcludeProfiles []Profile
	Disabled        bool

	Providers []any
	Values    []any
	Invokes   []any

	StartHooks  []any
	StopHooks   []any
	HealthHooks []any

	Modules []Module
	Tags    []string
}

type App struct {
	profile Profile
	meta    AppMeta

	container Container

	modules         []Module
	enabledModules  []Module
	disabledModules []Module

	invokes     []Callable
	startHooks  []Callable
	stopHooks   []Callable
	healthHooks []Callable

	state atomic.Int32
}