package render

import (
	"reflect"
	"strings"

	"github.com/DaiYuANg/archgo/sqltmplx/dialect"
	"github.com/samber/lo"
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
	v := reflect.ValueOf(params)
	for v.IsValid() && v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return map[string]any{}
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return map[string]any{}
	}
	if v.Kind() == reflect.Map {
		out := make(map[string]any, v.Len())
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
		fields := lo.Filter(lo.Range(t.NumField()), func(i int, _ int) bool {
			return t.Field(i).IsExported()
		})
		return lo.Assign(lo.Map(fields, func(i int, _ int) map[string]any {
			f := t.Field(i)
			val := v.Field(i).Interface()
			return map[string]any{
				f.Name:                 val,
				strings.ToLower(f.Name): val,
			}
		})...)
	}
	return map[string]any{}
}
