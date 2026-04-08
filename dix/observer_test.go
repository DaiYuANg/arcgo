package dix_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/DaiYuANg/arcgo/dix"
)

type recordingObserver struct {
	mu          sync.Mutex
	builds      []dix.BuildEvent
	starts      []dix.StartEvent
	stops       []dix.StopEvent
	health      []dix.HealthCheckEvent
	transitions []dix.StateTransitionEvent
}

func (r *recordingObserver) OnBuild(_ context.Context, event dix.BuildEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.builds = append(r.builds, event)
}

func (r *recordingObserver) OnStart(_ context.Context, event dix.StartEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.starts = append(r.starts, event)
}

func (r *recordingObserver) OnStop(_ context.Context, event dix.StopEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stops = append(r.stops, event)
}

func (r *recordingObserver) OnHealthCheck(_ context.Context, event dix.HealthCheckEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.health = append(r.health, event)
}

func (r *recordingObserver) OnStateTransition(_ context.Context, event dix.StateTransitionEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.transitions = append(r.transitions, event)
}

func TestObserverReceivesLifecycleEvents(t *testing.T) {
	observer := &recordingObserver{}
	app := dix.New("observer-app",
		dix.WithObserver(observer),
		dix.WithModule(
			dix.NewModule("health",
				dix.Setups(dix.Setup(func(c *dix.Container, _ dix.Lifecycle) error {
					c.RegisterHealthCheck("db", func(context.Context) error { return nil })
					return nil
				})),
				dix.Hooks(
					dix.OnStart0(func(context.Context) error { return nil }),
					dix.OnStop0(func(context.Context) error { return nil }),
				),
			),
		),
	)

	rt, err := app.Build()
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if err := rt.Start(context.Background()); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	report := rt.CheckHealth(context.Background())
	if !report.Healthy() {
		t.Fatalf("expected healthy report, got %v", report.Error())
	}

	if err := rt.Stop(context.Background()); err != nil {
		t.Fatalf("stop failed: %v", err)
	}

	if len(observer.builds) != 1 {
		t.Fatalf("expected 1 build event, got %d", len(observer.builds))
	}
	build := observer.builds[0]
	if build.Meta.Name != "observer-app" {
		t.Fatalf("expected build app name observer-app, got %q", build.Meta.Name)
	}
	if build.ModuleCount != 1 || build.SetupCount != 1 || build.HookCount != 2 {
		t.Fatalf("unexpected build counts: %+v", build)
	}
	if build.Err != nil {
		t.Fatalf("expected successful build event, got %v", build.Err)
	}

	if len(observer.starts) != 1 {
		t.Fatalf("expected 1 start event, got %d", len(observer.starts))
	}
	start := observer.starts[0]
	if start.StartHookCount != 1 || start.StartedHookCount != 1 {
		t.Fatalf("unexpected start counts: %+v", start)
	}
	if start.Err != nil {
		t.Fatalf("expected successful start event, got %v", start.Err)
	}

	if len(observer.health) != 1 {
		t.Fatalf("expected 1 health event, got %d", len(observer.health))
	}
	health := observer.health[0]
	if health.Kind != dix.HealthKindGeneral || health.Name != "db" {
		t.Fatalf("unexpected health event: %+v", health)
	}
	if health.Err != nil {
		t.Fatalf("expected successful health event, got %v", health.Err)
	}

	if len(observer.stops) != 1 {
		t.Fatalf("expected 1 stop event, got %d", len(observer.stops))
	}
	stop := observer.stops[0]
	if stop.StopHookCount != 1 {
		t.Fatalf("unexpected stop counts: %+v", stop)
	}
	if stop.Err != nil {
		t.Fatalf("expected successful stop event, got %v", stop.Err)
	}

	if len(observer.transitions) != 4 {
		t.Fatalf("expected 4 transitions, got %d", len(observer.transitions))
	}
	expected := []struct {
		from dix.AppState
		to   dix.AppState
	}{
		{from: dix.AppStateCreated, to: dix.AppStateBuilt},
		{from: dix.AppStateBuilt, to: dix.AppStateStarting},
		{from: dix.AppStateStarting, to: dix.AppStateStarted},
		{from: dix.AppStateStarted, to: dix.AppStateStopped},
	}
	for index, transition := range expected {
		got := observer.transitions[index]
		if got.From != transition.from || got.To != transition.to {
			t.Fatalf("unexpected transition at %d: %+v", index, got)
		}
	}
}

func TestObserverReceivesBuildFailureEvent(t *testing.T) {
	observer := &recordingObserver{}
	app := dix.New("observer-build-failure",
		dix.WithObserver(observer),
		dix.WithModule(
			dix.NewModule("broken",
				dix.WithModuleInvokes(dix.RawInvoke(func(*dix.Container) error {
					return errors.New("boom")
				})),
			),
		),
	)

	_, err := app.Build()
	if err == nil {
		t.Fatal("expected build failure")
	}

	if len(observer.builds) != 1 {
		t.Fatalf("expected 1 build event, got %d", len(observer.builds))
	}
	if observer.builds[0].Err == nil {
		t.Fatal("expected build event error to be set")
	}
}
