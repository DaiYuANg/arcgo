package validate

import (
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dbx/dialect"
)

type Validator interface {
	Validate(sql string) error
}

type Analyzer interface {
	Analyze(sql string) (*Analysis, error)
}

type SQLParser interface {
	Validator
	Analyzer
}

type Analysis struct {
	Dialect       string
	StatementType string
	NormalizedSQL string
	AST           any
}

type Func func(string) error

func (f Func) Validate(sql string) error { return f(sql) }

type Factory func() SQLParser

var parserRegistry = collectionx.NewConcurrentMap[string, Factory]()

func NewSQLParser(d dialect.Contract) SQLParser {
	name := strings.ToLower(strings.TrimSpace(d.Name()))
	if factory, ok := parserRegistry.Get(name); ok {
		return factory()
	}
	return &noopParser{dialect: d.Name()}
}

func Register(dialectName string, factory Factory) {
	if factory == nil {
		return
	}
	parserRegistry.Set(strings.ToLower(strings.TrimSpace(dialectName)), factory)
}

type noopParser struct{ dialect string }

func (n *noopParser) Validate(_ string) error { return nil }

func (n *noopParser) Analyze(sql string) (*Analysis, error) {
	return &Analysis{
		Dialect:       n.dialect,
		StatementType: detectStatementType(sql),
		NormalizedSQL: sql,
		AST:           nil,
	}, nil
}
