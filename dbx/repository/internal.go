package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/samber/lo"
	"github.com/samber/mo"
)

type countRow struct {
	Count int64 `dbx:"count"`
}

func (r *Base[E, S]) defaultSelect() *dbx.SelectQuery {
	items := collectionx.MapList(r.mapper.Fields(), func(_ int, field dbx.MappedField) dbx.SelectItem {
		return dbx.NamedColumn[any](r.schema, field.Column)
	})
	return dbx.SelectList(items).From(r.schema)
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
	cloned := query.Clone()
	if cloned == nil {
		return nil
	}
	cloned.Orders = nil
	cloned.LimitN = nil
	cloned.OffsetN = nil
	return cloned
}

func countQueryRequiresWrap(query *dbx.SelectQuery) bool {
	return query != nil &&
		(query.Distinct || query.Groups.Len() > 0 || query.HavingExp != nil || query.Unions.Len() > 0)
}

func wrappedCountBound(session dbx.Session, query *dbx.SelectQuery) (dbx.BoundQuery, error) {
	source := cloneForCount(query)
	bound, err := dbx.Build(session, source)
	if err != nil {
		return dbx.BoundQuery{}, fmt.Errorf("build wrapped count query: %w", err)
	}
	quotedCount := session.Dialect().QuoteIdent("count")
	quotedAlias := session.Dialect().QuoteIdent("dbx_count_source")
	return dbx.BoundQuery{
		SQL:          "SELECT COUNT(*) AS " + quotedCount + " FROM (" + bound.SQL + ") AS " + quotedAlias,
		Args:         bound.Args,
		CapacityHint: 1,
	}, nil
}

func firstCount(rows collectionx.List[countRow]) int64 {
	if rows.IsEmpty() {
		return 0
	}
	row, _ := rows.GetFirst()
	return row.Count
}

func cloneOrDefault[E any, S EntitySchema[E]](r *Base[E, S], query *dbx.SelectQuery) *dbx.SelectQuery {
	if query == nil {
		return r.defaultSelect()
	}
	return query.Clone()
}

func optionFromResult[T any](item T, err error) (mo.Option[T], error) {
	if err == nil {
		return mo.Some(item), nil
	}
	if errors.Is(err, ErrNotFound) {
		return mo.None[T](), nil
	}
	return mo.None[T](), err
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
		if primary, ok := provider.PrimaryKey(); ok && primary.Columns.Len() > 0 {
			return primary.Columns.Values()
		}
	}
	return []string{r.primaryColumnName()}
}

func keyPredicate[S dbx.TableSource](schema S, key Key) dbx.Predicate {
	if len(key) == 0 {
		return nil
	}
	columns := lo.Keys(key)
	slices.Sort(columns)
	predicates := lo.Map(columns, func(column string, _ int) dbx.Predicate {
		return dbx.NamedColumn[any](schema, column).Eq(key[column])
	})
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
