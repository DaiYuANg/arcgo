package search

import "github.com/DaiYuANg/arcgo/kvx"

// SchemaBuilder helps build search index schemas.
type SchemaBuilder struct {
	fields []kvx.SchemaField
}

// NewSchemaBuilder creates a new SchemaBuilder.
func NewSchemaBuilder() *SchemaBuilder {
	return &SchemaBuilder{
		fields: make([]kvx.SchemaField, 0),
	}
}

// TextField adds a text field to the schema.
func (sb *SchemaBuilder) TextField(name string, sortable bool) *SchemaBuilder {
	sb.fields = append(sb.fields, kvx.SchemaField{
		Name:     name,
		Type:     kvx.SchemaFieldTypeText,
		Indexing: true,
		Sortable: sortable,
	})
	return sb
}

// TagField adds a tag field to the schema.
func (sb *SchemaBuilder) TagField(name string, sortable bool) *SchemaBuilder {
	sb.fields = append(sb.fields, kvx.SchemaField{
		Name:     name,
		Type:     kvx.SchemaFieldTypeTag,
		Indexing: true,
		Sortable: sortable,
	})
	return sb
}

// NumericField adds a numeric field to the schema.
func (sb *SchemaBuilder) NumericField(name string, sortable bool) *SchemaBuilder {
	sb.fields = append(sb.fields, kvx.SchemaField{
		Name:     name,
		Type:     kvx.SchemaFieldTypeNumeric,
		Indexing: true,
		Sortable: sortable,
	})
	return sb
}

// Build builds the schema.
func (sb *SchemaBuilder) Build() []kvx.SchemaField {
	return sb.fields
}
