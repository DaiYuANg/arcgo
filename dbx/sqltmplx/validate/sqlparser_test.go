package validate

import "testing"

func TestNewSQLParser(t *testing.T) {
	t.Run("fallback to noop when backend is not registered", func(t *testing.T) {
		parserRegistry.Clear()
		parser := NewSQLParser(testDialect{name: "mysql"})
		if parser == nil {
			t.Fatal("parser should not be nil")
		}
		analysis, err := parser.Analyze("select 1")
		if err != nil {
			t.Fatalf("Analyze returned error: %v", err)
		}
		if analysis.Dialect != "mysql" {
			t.Fatalf("unexpected dialect: %q", analysis.Dialect)
		}
	})

	t.Run("registered backend is selected by dialect name", func(t *testing.T) {
		parserRegistry.Clear()
		t.Cleanup(func() {
			parserRegistry.Clear()
		})

		Register("postgres", func() SQLParser { return stubParser{} })

		parser := NewSQLParser(testDialect{name: "postgres"})
		if parser == nil {
			t.Fatal("parser should not be nil")
		}

		if _, ok := parser.(stubParser); !ok {
			t.Fatalf("unexpected parser type: %T", parser)
		}
	})
}

type testDialect struct {
	name string
}

func (d testDialect) BindVar(_ int) string { return "?" }
func (d testDialect) Name() string         { return d.name }

type stubParser struct{}

func (stubParser) Validate(string) error { return nil }

func (stubParser) Analyze(sql string) (*Analysis, error) {
	return &Analysis{
		Dialect:       "postgres",
		StatementType: detectStatementType(sql),
		NormalizedSQL: sql,
	}, nil
}
