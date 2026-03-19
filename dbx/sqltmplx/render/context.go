package render

import (
	"reflect"

	"github.com/DaiYuANg/arcgo/dbx/dialect"
	"github.com/samber/lo"
)

type state struct {
	dialect dialect.Contract
	params  any
	args    []any
	bindN   int
}

func newState(params any, d dialect.Contract) *state {
	return &state{dialect: d, params: params}
}

func (s *state) nextBind() string {
	s.bindN++
	return s.dialect.BindVar(s.bindN)
}

func exprEnv(params any) map[string]any {
	env := envMap(params)
	env["empty"] = isEmpty
	env["blank"] = isBlank
	env["present"] = isPresent
	return env
}

func envMap(params any) map[string]any {
	v, ok := indirectValue(params)
	if !ok {
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
		meta := cachedStructMetadata(v.Type())
		return lo.Assign(lo.Map(meta.fields, func(field structFieldMetadata, _ int) map[string]any {
			val := v.Field(field.index).Interface()
			out := map[string]any{
				field.name:       val,
				field.foldedName: val,
			}
			for _, alias := range field.aliases {
				out[alias] = val
			}
			return out
		})...)
	}
	return map[string]any{}
}
