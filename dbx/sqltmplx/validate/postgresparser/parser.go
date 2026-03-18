package postgresparser

import (
	"strings"

	"github.com/DaiYuANg/arcgo/dbx/sqltmplx/validate"
	pgquery "github.com/wasilibs/go-pgquery"
)

func init() {
	validate.Register("postgres", New)
}

type Parser struct{}

func New() validate.SQLParser { return &Parser{} }

func (p *Parser) Validate(sql string) error {
	_, err := pgquery.Parse(sql)
	return err
}

func (p *Parser) Analyze(sql string) (*validate.Analysis, error) {
	result, err := pgquery.Parse(sql)
	if err != nil {
		return nil, err
	}

	normalized, err := pgquery.Normalize(sql)
	if err != nil {
		normalized = normalizeWhitespace(sql)
	}

	return &validate.Analysis{
		Dialect:       "postgres",
		StatementType: detectStatementType(sql),
		NormalizedSQL: normalized,
		AST:           result,
	}, nil
}

func normalizeWhitespace(sql string) string {
	return strings.Join(strings.Fields(sql), " ")
}

func detectStatementType(sql string) string {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return "UNKNOWN"
	}
	parts := strings.Fields(sql)
	if len(parts) == 0 {
		return "UNKNOWN"
	}
	return strings.ToUpper(parts[0])
}
