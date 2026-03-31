package fx_test

import (
	"testing"

	pkgfx "github.com/DaiYuANg/arcgo/pkg/fx"
	uberfx "go.uber.org/fx"
)

func TestProvideOptionGroup(t *testing.T) {
	type config struct {
		Value int
	}

	type option func(*config)

	type params struct {
		uberfx.In

		Options []option `group:"test_options"`
	}

	var got []option
	app := uberfx.New(
		pkgfx.ProvideOptionGroup[config, option]("test_options",
			func(c *config) { c.Value = 1 },
			nil,
			func(c *config) { c.Value++ },
		),
		uberfx.Invoke(func(p params) {
			got = p.Options
		}),
	)
	defer func() {
		if err := app.Stop(t.Context()); err != nil {
			t.Fatalf("expected app stop to succeed, got %v", err)
		}
	}()

	if err := app.Err(); err != nil {
		t.Fatalf("expected valid app, got %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 grouped options, got %d", len(got))
	}

	cfg := config{}
	for _, opt := range got {
		opt(&cfg)
	}
	if cfg.Value != 2 {
		t.Fatalf("expected applied config value 2, got %d", cfg.Value)
	}
}

func TestProvideOptionGroupIgnoresEmptyGroup(t *testing.T) {
	app := uberfx.New(pkgfx.ProvideOptionGroup[struct{}, func(*struct{})]("", func(*struct{}) {}))
	defer func() {
		if err := app.Stop(t.Context()); err != nil {
			t.Fatalf("expected app stop to succeed, got %v", err)
		}
	}()

	if err := app.Err(); err != nil {
		t.Fatalf("expected valid app, got %v", err)
	}
}
