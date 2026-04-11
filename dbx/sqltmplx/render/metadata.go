package render

import (
	"reflect"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/samber/hot"
)

var structMetadataCache = hot.NewHotCache[reflect.Type, *structMetadata](hot.LRU, 256).Build()

type structMetadata struct {
	fields collectionx.List[structFieldMetadata]
	lookup collectionx.Map[string, structFieldMetadata]
}

type structFieldMetadata struct {
	index      int
	name       string
	foldedName string
	aliases    collectionx.List[string]
}

func cachedStructMetadata(t reflect.Type) *structMetadata {
	if cached, ok := structMetadataCache.Peek(t); ok {
		return cached
	}

	metadata := buildStructMetadata(t)
	if cached, ok := structMetadataCache.Peek(t); ok {
		return cached
	}
	structMetadataCache.Set(t, metadata)
	return metadata
}

func buildStructMetadata(t reflect.Type) *structMetadata {
	fields := collectionx.NewListWithCapacity[structFieldMetadata](t.NumField())
	for index := range t.NumField() {
		field := t.Field(index)
		if !field.IsExported() {
			continue
		}

		fields.Add(structFieldMetadata{
			index:      index,
			name:       field.Name,
			foldedName: strings.ToLower(field.Name),
			aliases:    fieldAliases(field),
		})
	}

	lookup := collectionx.NewMapWithCapacity[string, structFieldMetadata](fields.Len() * 3)
	fields.Range(func(_ int, field structFieldMetadata) bool {
		lookup.Set(field.foldedName, field)
		field.aliases.Range(func(_ int, alias string) bool {
			lookup.Set(strings.ToLower(alias), field)
			return true
		})
		return true
	})

	return &structMetadata{
		fields: fields,
		lookup: lookup,
	}
}

func indirectValue(input any) (reflect.Value, bool) {
	value := reflect.ValueOf(input)
	for value.IsValid() && value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return reflect.Value{}, false
		}
		value = value.Elem()
	}

	return value, value.IsValid()
}
