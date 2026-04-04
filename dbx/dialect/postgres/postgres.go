package postgres

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/DaiYuANg/arcgo/dbx/dialect"
)

// Dialect implements PostgreSQL rendering and schema inspection.
type Dialect struct{}

// New returns a PostgreSQL dialect implementation.
func New() Dialect { return Dialect{} }

// Name returns the dialect name.
func (Dialect) Name() string { return "postgres" }

// BindVar returns the bind placeholder for a parameter index.
func (Dialect) BindVar(n int) string { return "$" + strconv.Itoa(n) }

// QuoteIdent quotes an identifier for PostgreSQL.
func (Dialect) QuoteIdent(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}

// RenderLimitOffset renders a LIMIT/OFFSET clause for PostgreSQL.
func (Dialect) RenderLimitOffset(limit, offset *int) (string, error) {
	if limit == nil && offset == nil {
		return "", nil
	}
	if limit != nil && offset != nil {
		return fmt.Sprintf("LIMIT %d OFFSET %d", *limit, *offset), nil
	}
	if limit != nil {
		return fmt.Sprintf("LIMIT %d", *limit), nil
	}
	return fmt.Sprintf("OFFSET %d", *offset), nil
}

// QueryFeatures returns the supported query feature set.
func (Dialect) QueryFeatures() dialect.QueryFeatures {
	return dialect.DefaultQueryFeatures("postgres")
}

// BuildCreateTable builds a CREATE TABLE statement.
func (d Dialect) BuildCreateTable(spec dbx.TableSpec) (dbx.BoundQuery, error) {
	parts := collectionx.NewListWithCapacity[string](spec.Columns.Len() + spec.ForeignKeys.Len() + spec.Checks.Len() + 1)
	inlinePrimaryKey := singlePrimaryKeyColumn(spec.PrimaryKey)
	spec.Columns.Range(func(_ int, column dbx.ColumnMeta) bool {
		parts.Add(d.columnDDL(column, inlinePrimaryKey == column.Name, false))
		return true
	})
	if spec.PrimaryKey != nil && spec.PrimaryKey.Columns.Len() > 1 {
		parts.Add(d.primaryKeyDDL(*spec.PrimaryKey))
	}
	spec.ForeignKeys.Range(func(_ int, foreignKey dbx.ForeignKeyMeta) bool {
		parts.Add(d.foreignKeyDDL(foreignKey))
		return true
	})
	spec.Checks.Range(func(_ int, check dbx.CheckMeta) bool {
		parts.Add(d.checkDDL(check))
		return true
	})
	return dbx.BoundQuery{
		SQL: "CREATE TABLE IF NOT EXISTS " + d.QuoteIdent(spec.Name) + " (" + joinPostgresStrings(parts, ", ") + ")",
	}, nil
}

// BuildAddColumn builds an ALTER TABLE ADD COLUMN statement.
func (d Dialect) BuildAddColumn(table string, column dbx.ColumnMeta) (dbx.BoundQuery, error) {
	return dbx.BoundQuery{
		SQL: "ALTER TABLE " + d.QuoteIdent(table) + " ADD COLUMN " + d.columnDDL(column, false, true),
	}, nil
}

// BuildCreateIndex builds a CREATE INDEX statement.
func (d Dialect) BuildCreateIndex(index dbx.IndexMeta) (dbx.BoundQuery, error) {
	prefix := "CREATE INDEX IF NOT EXISTS "
	if index.Unique {
		prefix = "CREATE UNIQUE INDEX IF NOT EXISTS "
	}
	return dbx.BoundQuery{
		SQL: prefix + d.QuoteIdent(index.Name) + " ON " + d.QuoteIdent(index.Table) + " (" + d.joinQuotedIdentifiers(index.Columns) + ")",
	}, nil
}

// BuildAddForeignKey builds an ALTER TABLE ADD CONSTRAINT statement for a foreign key.
func (d Dialect) BuildAddForeignKey(table string, foreignKey dbx.ForeignKeyMeta) (dbx.BoundQuery, error) {
	return dbx.BoundQuery{
		SQL: "ALTER TABLE " + d.QuoteIdent(table) + " ADD " + d.foreignKeyDDL(foreignKey),
	}, nil
}

// BuildAddCheck builds an ALTER TABLE ADD CONSTRAINT statement for a check.
func (d Dialect) BuildAddCheck(table string, check dbx.CheckMeta) (dbx.BoundQuery, error) {
	return dbx.BoundQuery{
		SQL: "ALTER TABLE " + d.QuoteIdent(table) + " ADD " + d.checkDDL(check),
	}, nil
}

// InspectTable inspects a PostgreSQL table definition from system catalogs.
func (d Dialect) InspectTable(ctx context.Context, executor dbx.Executor, table string) (dbx.TableState, error) {
	exists, err := inspectPostgresTableExists(ctx, executor, table)
	if err != nil {
		return dbx.TableState{}, err
	}
	if !exists {
		return dbx.TableState{Name: table, Exists: false}, nil
	}

	primaryKey, primaryColumns, err := inspectPostgresPrimaryKey(ctx, executor, table)
	if err != nil {
		return dbx.TableState{}, err
	}

	columns, err := d.inspectColumns(ctx, executor, table, primaryColumns)
	if err != nil {
		return dbx.TableState{}, err
	}

	indexes, err := d.inspectIndexes(ctx, executor, table)
	if err != nil {
		return dbx.TableState{}, err
	}

	foreignKeys, err := d.inspectForeignKeys(ctx, executor, table)
	if err != nil {
		return dbx.TableState{}, err
	}

	checks, err := d.inspectChecks(ctx, executor, table)
	if err != nil {
		return dbx.TableState{}, err
	}

	return dbx.TableState{
		Exists:      true,
		Name:        table,
		Columns:     collectionx.NewListWithCapacity(len(columns), columns...),
		Indexes:     collectionx.NewListWithCapacity(len(indexes), indexes...),
		PrimaryKey:  primaryKey,
		ForeignKeys: collectionx.NewListWithCapacity(len(foreignKeys), foreignKeys...),
		Checks:      collectionx.NewListWithCapacity(len(checks), checks...),
	}, nil
}

// NormalizeType normalizes database type names into dbx logical types.
func (Dialect) NormalizeType(value string) string {
	typeName := strings.ToLower(strings.TrimSpace(value))
	if normalized, ok := postgresNormalizedTypes[typeName]; ok {
		return normalized
	}
	return typeName
}

var _ dbx.SchemaDialect = Dialect{}
