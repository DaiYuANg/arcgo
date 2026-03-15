package render

import (
	"reflect"
	"strings"

	"github.com/samber/lo"
	"github.com/samber/mo"
)

func lookup(params any, name string) mo.Option[any] {
	v := reflect.ValueOf(params)
	for v.IsValid() && v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return mo.None[any]()
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return mo.None[any]()
	}
	if v.Kind() == reflect.Map {
		mv := v.MapIndex(reflect.ValueOf(name))
		if mv.IsValid() {
			return mo.Some(mv.Interface())
		}
		return mo.None[any]()
	}
	if v.Kind() == reflect.Struct {
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			if f.Name == name || strings.EqualFold(f.Name, name) {
				return mo.Some(v.Field(i).Interface())
			}
		}
	}
	return mo.None[any]()
}

func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	for rv.IsValid() && rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return true
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() {
		return true
	}
	switch rv.Kind() {
	case reflect.String, reflect.Array, reflect.Slice, reflect.Map:
		return rv.Len() == 0
	}
	return false
}

// isBlank checks if value is nil or empty using lo.IsEmpty
func isBlank(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	for rv.IsValid() && rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return true
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() {
		return true
	}
	return lo.IsEmpty(rv.Interface())
}
