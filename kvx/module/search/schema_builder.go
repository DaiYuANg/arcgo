package search

import (
	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/kvx"
	"github.com/samber/lo"
)

// SchemaBuilder helps build search index schemas.
type SchemaBuilder struct {
	fields []kvx.SchemaField
}

// NewSchemaBuilder creates a new SchemaBuilder.
func NewSchemaBuilder() *SchemaBuilder {
	return &SchemaBuilder{}
}

// TextField adds a text field to the schema.
func (sb *SchemaBuilder) TextField(name string, sortable bool) *SchemaBuilder {
	sb.fields = lo.Concat(sb.fields, []kvx.SchemaField{{
		Name:     name,
		Type:     kvx.SchemaFieldTypeText,
		Indexing: true,
		Sortable: sortable,
	}})
	return sb
}

// TagField adds a tag field to the schema.
func (sb *SchemaBuilder) TagField(name string, sortable bool) *SchemaBuilder {
	sb.fields = lo.Concat(sb.fields, []kvx.SchemaField{{
		Name:     name,
		Type:     kvx.SchemaFieldTypeTag,
		Indexing: true,
		Sortable: sortable,
	}})
	return sb
}

// NumericField adds a numeric field to the schema.
func (sb *SchemaBuilder) NumericField(name string, sortable bool) *SchemaBuilder {
	sb.fields = lo.Concat(sb.fields, []kvx.SchemaField{{
		Name:     name,
		Type:     kvx.SchemaFieldTypeNumeric,
		Indexing: true,
		Sortable: sortable,
	}})
	return sb
}

// Build builds the schema.
func (sb *SchemaBuilder) Build() collectionx.List[kvx.SchemaField] {
	return collectionx.NewListWithCapacity(len(sb.fields), sb.fields...)
}
