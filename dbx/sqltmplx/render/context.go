package render

import (
	"reflect"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dbx/dialect"
)

type state struct {
	dialect dialect.Contract
	params  any
	args    collectionx.List[any]
	bindN   int
}

func newState(params any, d dialect.Contract) *state {
	return &state{dialect: d, params: params, args: collectionx.NewList[any]()}
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
		return mapEnv(v)
	}
	if v.Kind() == reflect.Struct {
		return structEnv(v)
	}
	return map[string]any{}
}

func mapEnv(value reflect.Value) map[string]any {
	out := make(map[string]any, value.Len())
	iter := value.MapRange()
	for iter.Next() {
		key := iter.Key()
		if key.Kind() == reflect.String {
			out[key.String()] = iter.Value().Interface()
		}
	}
	return out
}

func structEnv(value reflect.Value) map[string]any {
	meta := cachedStructMetadata(value.Type())
	out := make(map[string]any, meta.fields.Len()*2)
	meta.fields.Range(func(_ int, field structFieldMetadata) bool {
		assignStructField(out, field, value.Field(field.index).Interface())
		return true
	})
	return out
}

func assignStructField(out map[string]any, field structFieldMetadata, value any) {
	out[field.name] = value
	out[field.foldedName] = value
	field.aliases.Range(func(_ int, alias string) bool {
		out[alias] = value
		return true
	})
}
