package dbx

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
)

type scanPlan struct {
	fields []MappedField
}

func (m StructMapper[E]) ScanRows(rows *sql.Rows) ([]E, error) {
	if m.meta == nil {
		return nil, ErrNilMapper
	}
	if rows == nil {
		return nil, fmt.Errorf("dbx: rows is nil")
	}

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	plan, err := m.scanPlan(columns)
	if err != nil {
		return nil, err
	}

	items := collectionx.NewList[E]()
	for rows.Next() {
		entity, err := m.scanCurrentRow(rows, plan)
		if err != nil {
			return nil, err
		}
		items.Add(entity)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items.Values(), nil
}

func (m StructMapper[E]) scanPlan(columns []string) (*scanPlan, error) {
	signature := scanSignature(columns)
	if cached, ok := m.meta.scanPlans.Get(signature); ok {
		return cached, nil
	}

	fields := collectionx.NewListWithCapacity[MappedField](len(columns))
	for _, column := range columns {
		field, ok := m.resolveFieldByResultColumn(column)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrUnmappedColumn, column)
		}
		fields.Add(field)
	}

	plan := &scanPlan{fields: fields.Values()}
	actual, _ := m.meta.scanPlans.GetOrStore(signature, plan)
	return actual, nil
}

func (m StructMapper[E]) scanCurrentRow(rows *sql.Rows, plan *scanPlan) (E, error) {
	value := reflect.New(m.meta.entityType).Elem()
	destinations := make([]any, len(plan.fields))
	codecSources := make([]any, len(plan.fields))
	for i, field := range plan.fields {
		fieldValue, err := ensureFieldValue(value, field)
		if err != nil {
			var zero E
			return zero, err
		}
		if field.codec != nil {
			destinations[i] = &codecSources[i]
			continue
		}
		destinations[i] = fieldValue.Addr().Interface()
	}

	if err := rows.Scan(destinations...); err != nil {
		var zero E
		return zero, err
	}
	for i, field := range plan.fields {
		if field.codec == nil {
			continue
		}
		fieldValue, err := ensureFieldValue(value, field)
		if err != nil {
			var zero E
			return zero, err
		}
		if err := field.codec.Decode(codecSources[i], fieldValue); err != nil {
			var zero E
			return zero, err
		}
	}
	return value.Interface().(E), nil
}

func (m StructMapper[E]) resolveFieldByResultColumn(column string) (MappedField, bool) {
	if m.meta == nil {
		return MappedField{}, false
	}
	if field, ok := m.meta.byColumn.Get(column); ok {
		return field, true
	}
	normalized := normalizeResultColumnName(column)
	if normalized == "" {
		return MappedField{}, false
	}
	return m.meta.byNormalizedColumn.Get(normalized)
}

func scanSignature(columns []string) string {
	return strings.Join(columns, "\x1f")
}

func normalizeResultColumnName(column string) string {
	trimmed := strings.TrimSpace(column)
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, ".")
	last := parts[len(parts)-1]
	last = strings.TrimSpace(last)
	last = strings.Trim(last, "`\"")
	last = strings.TrimPrefix(last, "[")
	last = strings.TrimSuffix(last, "]")
	return strings.ToLower(strings.TrimSpace(last))
}
