// Package adapter provides client adapters for Redis and Valkey.
package kvx

import (
	"errors"
)

// Common errors that adapters should convert to.
var (
	ErrNil = errors.New("kvx: nil") // Key not found error
)

// IsNil checks if the error is a "not found" error.
func IsNil(err error) bool {
	return errors.Is(err, ErrNil)
}

// SchemaFieldType mapping for adapters.
type SchemaFieldType string

const (
	SchemaFieldTypeText    SchemaFieldType = "TEXT"
	SchemaFieldTypeTag     SchemaFieldType = "TAG"
	SchemaFieldTypeNumeric SchemaFieldType = "NUMERIC"
)

// ConvertSchemaFields converts kvx schema fields to adapter schema fields.
func ConvertSchemaFields(fields []SchemaField) []SchemaField {
	result := make([]SchemaField, len(fields))
	for i, f := range fields {
		result[i] = SchemaField{
			Name:     f.Name,
			Type:     SchemaFieldType(f.Type),
			Indexing: f.Indexing,
			Sortable: f.Sortable,
		}
	}
	return result
}

// SchemaField represents a search schema field for adapters.
type SchemaField struct {
	Name     string
	Type     SchemaFieldType
	Indexing bool
	Sortable bool
}
