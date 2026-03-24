package activerecord

import (
	"fmt"
	"reflect"

	"github.com/DaiYuANg/arcgo/dbx"
	"github.com/DaiYuANg/arcgo/dbx/repository"
	"github.com/samber/lo"
)

func (s *Store[E, S]) keyOf(entity *E) repository.Key {
	if entity == nil {
		return nil
	}
	columns := primaryKeyColumns(s.repository.Schema())
	if len(columns) == 0 {
		return nil
	}
	root := reflect.ValueOf(entity)
	if root.Kind() != reflect.Ptr || root.IsNil() {
		return nil
	}
	root = root.Elem()
	if root.Kind() != reflect.Struct {
		return nil
	}
	key := make(repository.Key, len(columns))
	for _, column := range columns {
		field, ok := s.repository.Mapper().FieldByColumn(column)
		if !ok {
			return nil
		}
		value, err := mappedFieldValue(root, field)
		if err != nil {
			return nil
		}
		key[column] = value.Interface()
	}
	return key
}

func primaryKeyColumns[S dbx.TableSource](schema S) []string {
	type primaryKeyProvider interface {
		PrimaryKey() (dbx.PrimaryKeyMeta, bool)
	}
	if provider, ok := any(schema).(primaryKeyProvider); ok {
		if primary, ok := provider.PrimaryKey(); ok && len(primary.Columns) > 0 {
			return append([]string(nil), primary.Columns...)
		}
	}
	type primaryColumnProvider interface {
		PrimaryColumn() (dbx.ColumnMeta, bool)
	}
	if provider, ok := any(schema).(primaryColumnProvider); ok {
		if column, ok := provider.PrimaryColumn(); ok && column.Name != "" {
			return []string{column.Name}
		}
	}
	return []string{"id"}
}

func mappedFieldValue(root reflect.Value, field dbx.MappedField) (reflect.Value, error) {
	value := root
	for _, index := range field.Path {
		if value.Kind() == reflect.Ptr {
			if value.IsNil() {
				return reflect.Value{}, fmt.Errorf("dbx: nil pointer for field %s", field.Name)
			}
			value = value.Elem()
		}
		if value.Kind() != reflect.Struct {
			return reflect.Value{}, fmt.Errorf("dbx: field %s path reaches non-struct", field.Name)
		}
		value = value.Field(index)
	}
	for value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return reflect.Zero(value.Type().Elem()), nil
		}
		value = value.Elem()
	}
	return value, nil
}

func cloneKey(key repository.Key) repository.Key {
	if len(key) == 0 {
		return nil
	}
	return lo.Assign(repository.Key{}, key)
}

func hasZeroKeyValue(key repository.Key) bool {
	for _, value := range key {
		if value == nil {
			return true
		}
		rv := reflect.ValueOf(value)
		if !rv.IsValid() {
			return true
		}
		switch rv.Kind() {
		case reflect.Ptr, reflect.Interface:
			if rv.IsNil() {
				return true
			}
		}
		if rv.IsZero() {
			return true
		}
	}
	return false
}

