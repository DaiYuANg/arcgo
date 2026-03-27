package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dbx"
)

func scanPostgresPrimaryKey(rows *sql.Rows) (string, string, error) {
	var name string
	var column string

	if err := rows.Scan(&name, &column); err != nil {
		return "", "", fmt.Errorf("scan postgres primary key: %w", err)
	}
	return name, column, nil
}

func scanPostgresColumn(rows *sql.Rows) (dbx.ColumnState, error) {
	var name string
	var udtName string
	var isNullable string
	var defaultValue sql.NullString
	var isIdentity bool

	if err := rows.Scan(&name, &udtName, &isNullable, &defaultValue, &isIdentity); err != nil {
		return dbx.ColumnState{}, fmt.Errorf("scan postgres column: %w", err)
	}

	return dbx.ColumnState{
		Name:          name,
		Type:          udtName,
		Nullable:      strings.EqualFold(isNullable, "YES"),
		AutoIncrement: isIdentity || strings.Contains(strings.ToLower(defaultValue.String), "nextval"),
		DefaultValue:  defaultValue.String,
	}, nil
}

func scanPostgresIndex(rows *sql.Rows) (dbx.IndexState, bool, error) {
	var name string
	var definition string

	if err := rows.Scan(&name, &definition); err != nil {
		return dbx.IndexState{}, false, fmt.Errorf("scan postgres index: %w", err)
	}

	upperDefinition := strings.ToUpper(definition)
	if strings.Contains(upperDefinition, "PRIMARY KEY") {
		return dbx.IndexState{}, true, nil
	}

	return dbx.IndexState{
		Name:    name,
		Columns: parseIndexColumns(definition),
		Unique:  strings.Contains(upperDefinition, "CREATE UNIQUE INDEX"),
	}, false, nil
}

func scanPostgresForeignKey(rows *sql.Rows) (string, dbx.ForeignKeyState, error) {
	var name string
	var column string
	var targetTable string
	var targetColumn string
	var updateRule string
	var deleteRule string

	if err := rows.Scan(&name, &column, &targetTable, &targetColumn, &updateRule, &deleteRule); err != nil {
		return "", dbx.ForeignKeyState{}, fmt.Errorf("scan postgres foreign key: %w", err)
	}

	return name, dbx.ForeignKeyState{
		Name:          name,
		TargetTable:   targetTable,
		Columns:       []string{column},
		TargetColumns: []string{targetColumn},
		OnDelete:      referentialAction(deleteRule),
		OnUpdate:      referentialAction(updateRule),
	}, nil
}

func scanPostgresCheck(rows *sql.Rows) (dbx.CheckState, error) {
	var name string
	var clause string

	if err := rows.Scan(&name, &clause); err != nil {
		return dbx.CheckState{}, fmt.Errorf("scan postgres check: %w", err)
	}

	return dbx.CheckState{Name: name, Expression: clause}, nil
}

func postgresPrimaryKeyState(name string, columns []string) *dbx.PrimaryKeyState {
	if len(columns) == 0 {
		return nil
	}
	return &dbx.PrimaryKeyState{Name: name, Columns: columns}
}

func postgresPrimaryColumn(columns map[string]struct{}, name string) bool {
	_, ok := columns[name]
	return ok
}

func appendPostgresForeignKey(groups collectionx.OrderedMap[string, dbx.ForeignKeyState], name string, state dbx.ForeignKeyState) {
	current, ok := groups.Get(name)
	if !ok {
		groups.Set(name, state)
		return
	}
	current.Columns = append(current.Columns, state.Columns...)
	current.TargetColumns = append(current.TargetColumns, state.TargetColumns...)
	groups.Set(name, current)
}

func queryPostgresRows(ctx context.Context, executor dbx.Executor, action, query string, args ...any) (*sql.Rows, error) {
	rows, err := executor.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", action, err)
	}
	return rows, nil
}

func closePostgresRows(action string, rows *sql.Rows) error {
	if rows == nil {
		return nil
	}
	if closeErr := rows.Close(); closeErr != nil {
		return fmt.Errorf("%s: close rows: %w", action, closeErr)
	}
	return nil
}

func postgresRowsError(action string, rows *sql.Rows) error {
	if err := rows.Err(); err != nil {
		return fmt.Errorf("%s: rows err: %w", action, err)
	}
	return nil
}
