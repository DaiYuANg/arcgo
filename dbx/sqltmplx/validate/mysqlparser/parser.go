package mysqlparser

import (
	"strings"

	"github.com/DaiYuANg/arcgo/dbx/sqltmplx/validate"
	"vitess.io/vitess/go/vt/sqlparser"
)

func init() {
	validate.Register("mysql", New)
}

type Parser struct {
	parser *sqlparser.Parser
}

func New() validate.SQLParser {
	return &Parser{parser: sqlparser.NewTestParser()}
}

func (p *Parser) Validate(sql string) error {
	_, err := p.parser.Parse(sql)
	return err
}

func (p *Parser) Analyze(sql string) (*validate.Analysis, error) {
	stmt, err := p.parser.Parse(sql)
	if err != nil {
		return nil, err
	}

	return &validate.Analysis{
		Dialect:       "mysql",
		StatementType: statementType(stmt),
		NormalizedSQL: normalizeWhitespace(sql),
		AST:           stmt,
	}, nil
}

func statementType(stmt sqlparser.Statement) string {
	switch stmt.(type) {
	case *sqlparser.Select:
		return "SELECT"
	case *sqlparser.Insert:
		return "INSERT"
	case *sqlparser.Update:
		return "UPDATE"
	case *sqlparser.Delete:
		return "DELETE"
	default:
		return detectStatementType(sqlparser.String(stmt))
	}
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
