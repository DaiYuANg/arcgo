package repository

import (
	"database/sql"
	"strings"

	"github.com/DaiYuANg/arcgo/dbx"
)

type countRow struct {
	Count int64 `dbx:"count"`
}

func (r *Base[E, S]) defaultSelect() *dbx.SelectQuery {
	fields := r.mapper.Fields()
	items := make([]dbx.SelectItem, 0, len(fields))
	for _, field := range fields {
		items = append(items, dbx.NamedColumn[any](r.schema, field.Column))
	}
	return dbx.Select(items...).From(r.schema)
}

func (r *Base[E, S]) applySpecs(specs ...Spec) *dbx.SelectQuery {
	query := r.defaultSelect()
	for _, spec := range specs {
		if spec != nil {
			query = spec.Apply(query)
		}
	}
	return query
}

func cloneForCount(query *dbx.SelectQuery) *dbx.SelectQuery {
	cloned := *query
	cloned.Orders = nil
	cloned.LimitN = nil
	cloned.OffsetN = nil
	return &cloned
}

func (r *Base[E, S]) primaryColumnName() string {
	type primaryColumnProvider interface {
		PrimaryColumn() (dbx.ColumnMeta, bool)
	}
	if provider, ok := any(r.schema).(primaryColumnProvider); ok {
		if column, ok := provider.PrimaryColumn(); ok && column.Name != "" {
			return column.Name
		}
	}
	return "id"
}

func (r *Base[E, S]) primaryKeyColumns() []string {
	type primaryKeyProvider interface {
		PrimaryKey() (dbx.PrimaryKeyMeta, bool)
	}
	if provider, ok := any(r.schema).(primaryKeyProvider); ok {
		if primary, ok := provider.PrimaryKey(); ok && len(primary.Columns) > 0 {
			return append([]string(nil), primary.Columns...)
		}
	}
	return []string{r.primaryColumnName()}
}

func keyPredicate[S dbx.TableSource](schema S, key Key) dbx.Predicate {
	if len(key) == 0 {
		return nil
	}
	predicates := make([]dbx.Predicate, 0, len(key))
	for column, value := range key {
		predicates = append(predicates, dbx.NamedColumn[any](schema, column).Eq(value))
	}
	return dbx.And(predicates...)
}

func hasAffectedRows(result sql.Result) bool {
	if result == nil {
		return false
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false
	}
	return rows > 0
}

func wrapMutationError(err error) error {
	if err == nil {
		return nil
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "unique") || strings.Contains(message, "duplicate") || strings.Contains(message, "constraint") {
		return &ConflictError{Err: err}
	}
	return err
}

