package option_test

import (
	"testing"

	"github.com/DaiYuANg/arcgo/pkg/option"
)

func TestApply(t *testing.T) {
	type config struct {
		Value int
		Name  string
	}

	cfg := config{}
	option.Apply(&cfg,
		func(c *config) { c.Value = 1 },
		nil,
		func(c *config) { c.Name = "arcgo" },
		func(c *config) { c.Value++ },
	)

	if cfg.Value != 2 {
		t.Fatalf("expected value 2, got %d", cfg.Value)
	}
	if cfg.Name != "arcgo" {
		t.Fatalf("expected name arcgo, got %q", cfg.Name)
	}
}

func TestApplyNilTarget(t *testing.T) {
	type config struct {
		Called bool
	}

	called := false
	option.Apply((*config)(nil), func(c *config) {
		called = true
	})

	if called {
		t.Fatal("expected nil target to skip option application")
	}
}
