package sqliteparser

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/DaiYuANg/arcgo/dbx/sqltmplx/validate"
	rqlitesql "github.com/rqlite/sql"
	_ "modernc.org/sqlite"
)

func init() {
	validate.Register("sqlite", New)
}

type Parser struct{}

func New() validate.SQLParser { return &Parser{} }

func (p *Parser) Validate(sqlText string) error {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare(sqlText)
	if stmt != nil {
		_ = stmt.Close()
	}
	return err
}

func (p *Parser) Analyze(sqlText string) (*validate.Analysis, error) {
	parser := rqlitesql.NewParser(strings.NewReader(sqlText))
	astNode, err := parser.ParseStatement()
	if err != nil {
		return nil, err
	}
	if err := p.Validate(sqlText); err != nil {
		return nil, fmt.Errorf("sqlite engine validation failed after AST parse: %w", err)
	}
	return &validate.Analysis{
		Dialect:       "sqlite",
		StatementType: detectStatementType(sqlText),
		NormalizedSQL: normalizeWhitespace(sqlText),
		AST:           astNode,
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
