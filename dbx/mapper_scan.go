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
	return m.scanRowsWithCapacity(rows, 0)
}

func (m StructMapper[E]) ScanRowsWithCapacity(rows *sql.Rows, capacityHint int) ([]E, error) {
	return m.scanRowsWithCapacity(rows, capacityHint)
}

func (m StructMapper[E]) scanRowsWithCapacity(rows *sql.Rows, capacityHint int) ([]E, error) {
	if m.meta == nil {
		return nil, ErrNilMapper
	}
	if rows == nil {
		return nil, errors.New("dbx: rows is nil")
	}

	columns, err := rows.Columns()
	if err != nil {
		return nil, wrapDBError("read row columns", err)
	}
	plan, err := m.scanPlan(columns)
	if err != nil {
		return nil, err
	}
	if capacityHint > 0 {
		return m.collectRowsWithCapacity(context.Background(), plan, rows, capacityHint)
	}
	items, err := scanlib.AllFromRows[E](context.Background(), m.scanMapper(plan), rows)
	return items, wrapDBError("scan all rows", err)
}

func (m StructMapper[E]) collectRowsWithCapacity(ctx context.Context, plan *scanPlan, rows *sql.Rows, capacityHint int) (_ []E, err error) {
	cursor, err := scanlib.CursorFromRows(ctx, m.scanMapper(plan), rows)
	if err != nil {
		return nil, wrapDBError("open scan cursor", err)
	}
	defer func() {
		err = errors.Join(err, wrapDBError("close scan cursor", cursor.Close()))
	}()
	result := make([]E, 0, capacityHint)
	for cursor.Next() {
		value, getErr := cursor.Get()
		if getErr != nil {
			return nil, wrapDBError("get scan cursor value", getErr)
		}
		result = append(result, value)
	}
	return result, wrapDBError("read scan cursor error", cursor.Err())
}

func (m StructMapper[E]) scanOneRows(ctx context.Context, rows *sql.Rows) (E, bool, error) {
	if m.meta == nil {
		var zero E
		return zero, false, ErrNilMapper
	}
	if rows == nil {
		var zero E
		return zero, false, errors.New("dbx: rows is nil")
	}

	columns, err := rows.Columns()
	if err != nil {
		var zero E
		return zero, false, wrapDBError("read row columns", err)
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
		return zero, false, wrapDBError("scan one row", err)
	}

	if rows.Next() {
		var zero E
		return zero, false, ErrTooManyRows
	}
	if err := rows.Err(); err != nil {
		var zero E
		return zero, false, wrapDBError("iterate rows", err)
	}

	return value, true, nil
}

func (m StructMapper[E]) scanCursor(ctx context.Context, rows *sql.Rows) (Cursor[E], error) {
	if m.meta == nil {
		return nil, ErrNilMapper
	}
	if rows == nil {
		return nil, errors.New("dbx: rows is nil")
	}

	columns, err := rows.Columns()
	if err != nil {
		return nil, wrapDBError("read row columns", err)
	}
	plan, err := m.scanPlan(columns)
	if err != nil {
		return nil, err
	}

	cursor, err := scanlib.CursorFromRows(ctx, m.scanMapper(plan), rows)
	if err != nil {
		return nil, wrapDBError("open scan cursor", err)
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
			return nil, &UnmappedColumnError{Column: column}
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
				for i := range plan.fields {
					field := plan.fields[i]
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
				current, ok := state.(rowScanState)
				if !ok {
					var zero E
					return zero, fmt.Errorf("dbx: unexpected scan state %T", state)
				}
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
						return zero, wrapDBError("decode mapped field", err)
					}
				}
				value, ok := current.value.Interface().(E)
				if !ok {
					var zero E
					return zero, fmt.Errorf("dbx: scanned value type %T does not match target", current.value.Interface())
				}
				return value, nil
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
