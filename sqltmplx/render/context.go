package render

import (
	"reflect"
	"strings"

	"github.com/DaiYuANg/archgo/sqltmplx/dialect"
)

type state struct {
	dialect dialect.Dialect
	params  any
	args    []any
	bindN   int
}

func newState(params any, d dialect.Dialect) *state {
	return &state{dialect: d, params: params}
}

func (s *state) nextBind() string {
	s.bindN++
	return s.dialect.BindVar(s.bindN)
}

func envMap(params any) map[string]any {
	out := map[string]any{}
	v := reflect.ValueOf(params)
	for v.IsValid() && v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return out
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return out
	}
	if v.Kind() == reflect.Map {
		iter := v.MapRange()
		for iter.Next() {
			k := iter.Key()
			if k.Kind() == reflect.String {
				out[k.String()] = iter.Value().Interface()
			}
		}
		return out
	}
	if v.Kind() == reflect.Struct {
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			out[f.Name] = v.Field(i).Interface()
			out[strings.ToLower(f.Name)] = v.Field(i).Interface()
		}
	}
	return out
}
