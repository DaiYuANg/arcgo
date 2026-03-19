package dbx

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	scanlib "github.com/stephenafamo/scan"
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
	return scanlib.AllFromRows[E](context.Background(), m.scanMapper(plan), rows)
}

func (m StructMapper[E]) scanOneRows(ctx context.Context, rows *sql.Rows) (E, bool, error) {
	if m.meta == nil {
		var zero E
		return zero, false, ErrNilMapper
	}
	if rows == nil {
		var zero E
		return zero, false, fmt.Errorf("dbx: rows is nil")
	}

	columns, err := rows.Columns()
	if err != nil {
		var zero E
		return zero, false, err
	}
	plan, err := m.scanPlan(columns)
	if err != nil {
		var zero E
		return zero, false, err
	}

	value, err := scanlib.OneFromRows[E](ctx, m.scanMapper(plan), rows)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			var zero E
			return zero, false, nil
		}
		var zero E
		return zero, false, err
	}

	if rows.Next() {
		var zero E
		return zero, false, ErrTooManyRows
	}
	if err := rows.Err(); err != nil {
		var zero E
		return zero, false, err
	}

	return value, true, nil
}

func (m StructMapper[E]) scanCursor(ctx context.Context, rows *sql.Rows) (Cursor[E], error) {
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

	cursor, err := scanlib.CursorFromRows(ctx, m.scanMapper(plan), rows)
	if err != nil {
		return nil, err
	}
	return scanCursor[E]{cursor: cursor}, nil
}

func (m StructMapper[E]) scanPlan(columns []string) (*scanPlan, error) {
	signature := scanSignature(columns)
	if cached, ok := m.meta.scanPlans.Peek(signature); ok {
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
	if cached, ok := m.meta.scanPlans.Peek(signature); ok {
		return cached, nil
	}
	m.meta.scanPlans.Set(signature, plan)
	return plan, nil
}

type rowScanState struct {
	value        reflect.Value
	codecSources []any
}

func (m StructMapper[E]) scanMapper(plan *scanPlan) scanlib.Mapper[E] {
	return func(_ context.Context, _ []string) (func(*scanlib.Row) (any, error), func(any) (E, error)) {
		return func(row *scanlib.Row) (any, error) {
				state := rowScanState{
					value:        reflect.New(m.meta.entityType).Elem(),
					codecSources: make([]any, len(plan.fields)),
				}
				for i, field := range plan.fields {
					fieldValue, err := ensureFieldValue(state.value, field)
					if err != nil {
						return nil, err
					}
					if field.codec != nil {
						row.ScheduleScanByIndexX(i, reflect.ValueOf(&state.codecSources[i]))
						continue
					}
					row.ScheduleScanByIndexX(i, fieldValue.Addr())
				}
				return state, nil
			}, func(state any) (E, error) {
				current := state.(rowScanState)
				for i, field := range plan.fields {
					if field.codec == nil {
						continue
					}
					fieldValue, err := ensureFieldValue(current.value, field)
					if err != nil {
						var zero E
						return zero, err
					}
					if err := field.codec.Decode(current.codecSources[i], fieldValue); err != nil {
						var zero E
						return zero, err
					}
				}
				return current.value.Interface().(E), nil
			}
	}
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
