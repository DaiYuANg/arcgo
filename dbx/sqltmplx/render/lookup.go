package render

import (
	"reflect"
	"strings"

	"github.com/samber/lo"
	"github.com/samber/mo"
)

func lookup(params any, name string) mo.Option[any] {
	cur := params
	remaining := name
	for remaining != "" {
		part, rest, found := strings.Cut(remaining, ".")
		remaining = rest

		next := lookupOne(cur, part)
		if next.IsAbsent() {
			return mo.None[any]()
		}
		cur = next.MustGet()
		if !found {
			break
		}
	}
	return mo.Some(cur)
}

func lookupOne(params any, name string) mo.Option[any] {
	v, ok := indirectValue(params)
	if !ok {
		return mo.None[any]()
	}
	if v.Kind() == reflect.Map {
		for _, key := range []string{name, strings.ToLower(name), strings.ToUpper(name)} {
			mv := v.MapIndex(reflect.ValueOf(key))
			if mv.IsValid() {
				return mo.Some(mv.Interface())
			}
		}
		return mo.None[any]()
	}
	if v.Kind() == reflect.Struct {
		meta := cachedStructMetadata(v.Type())
		if field, exists := meta.lookup[strings.ToLower(name)]; exists {
			return mo.Some(v.Field(field.index).Interface())
		}
	}
	return mo.None[any]()
}

func fieldAliases(f reflect.StructField) []string {
	return lo.Uniq(lo.FlatMap([]string{"sqltmpl", "db", "json"}, func(tagKey string, _ int) []string {
		raw := strings.TrimSpace(f.Tag.Get(tagKey))
		if raw == "" || raw == "-" {
			return nil
		}
		alias := strings.TrimSpace(strings.Split(raw, ",")[0])
		if alias == "" || alias == "-" {
			return nil
		}
		return []string{alias}
	}))
}

func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	rv, ok := indirectValue(v)
	if !ok {
		return true
	}
	switch rv.Kind() {
	case reflect.String, reflect.Array, reflect.Slice, reflect.Map:
		return rv.Len() == 0
	}
	return false
}

func isBlank(v any) bool {
	if v == nil {
		return true
	}
	rv, ok := indirectValue(v)
	if !ok {
		return true
	}
	return lo.IsEmpty(rv.Interface())
}

func isPresent(v any) bool {
	return !isBlank(v)
}
