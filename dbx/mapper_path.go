package dbx

import (
	"fmt"
	"reflect"
)

func appendIndexPath(prefix []int, index int) []int {
	path := make([]int, len(prefix)+1)
	copy(path, prefix)
	path[len(prefix)] = index
	return path
}

func indirectStructType(typ reflect.Type) (reflect.Type, bool) {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, false
	}
	return typ, true
}

func fieldPath(field MappedField) []int {
	if len(field.Path) > 0 {
		return field.Path
	}
	return []int{field.Index}
}

func ensureFieldValue(root reflect.Value, field MappedField) (reflect.Value, error) {
	current := root
	path := fieldPath(field)
	for i, index := range path {
		current = current.Field(index)
		if i == len(path)-1 {
			return current, nil
		}
		for current.Kind() == reflect.Pointer {
			if current.IsNil() {
				current.Set(reflect.New(current.Type().Elem()))
			}
			current = current.Elem()
		}
		if current.Kind() != reflect.Struct {
			return reflect.Value{}, fmt.Errorf("dbx: field path %v does not resolve to struct", path[:i+1])
		}
	}
	return reflect.Value{}, fmt.Errorf("dbx: field path %v is empty", path)
}

func fieldValueForRead(root reflect.Value, field MappedField) (reflect.Value, error) {
	current := root
	path := fieldPath(field)
	for i, index := range path {
		current = current.Field(index)
		if i == len(path)-1 {
			return current, nil
		}
		for current.Kind() == reflect.Pointer {
			if current.IsNil() {
				if field.Type == nil {
					return reflect.Value{}, fmt.Errorf("dbx: field %s type metadata is missing", field.Name)
				}
				return reflect.Zero(field.Type), nil
			}
			current = current.Elem()
		}
		if current.Kind() != reflect.Struct {
			return reflect.Value{}, fmt.Errorf("dbx: field path %v does not resolve to struct", path[:i+1])
		}
	}
	return reflect.Value{}, fmt.Errorf("dbx: field path %v is empty", path)
}

func normalizeFieldValue(value reflect.Value) any {
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Pointer && value.IsNil() {
		return nil
	}
	return value.Interface()
}

func boundFieldValue(field MappedField, value reflect.Value) (any, error) {
	if field.codec == nil {
		return normalizeFieldValue(value), nil
	}
	return field.codec.Encode(value)
}
