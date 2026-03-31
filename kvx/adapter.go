package kvx

import (
	"errors"

	"github.com/samber/lo"
)

// Common errors that adapters should convert to.
var (
	ErrNil                  = errors.New("kvx: nil") // Key not found error
	ErrTooManyArgs          = errors.New("too many redis pipeline args")
	ErrInvalidClientOptions = errors.New("kvx: invalid client options")
	ErrUnsupportedOption    = errors.New("kvx: unsupported client option")
)

// MaxPipelineArgs is the maximum number of arguments allowed in a single pipeline command.
const MaxPipelineArgs = 1024

// IsNil checks if the error is a "not found" error.
func IsNil(err error) bool {
	return errors.Is(err, ErrNil)
}

// SchemaFieldType mapping for adapters.
type SchemaFieldType string

const (
	// SchemaFieldTypeText indexes a field as full text.
	SchemaFieldTypeText SchemaFieldType = "TEXT"
	// SchemaFieldTypeTag indexes a field as an exact-match tag.
	SchemaFieldTypeTag SchemaFieldType = "TAG"
	// SchemaFieldTypeNumeric indexes a field as a numeric value.
	SchemaFieldTypeNumeric SchemaFieldType = "NUMERIC"
)

// ConvertSchemaFields converts kvx schema fields to adapter schema fields.
func ConvertSchemaFields(fields []SchemaField) []SchemaField {
	return lo.Map(fields, func(f SchemaField, _ int) SchemaField {
		return SchemaField{
			Name:     f.Name,
			Type:     f.Type,
			Indexing: f.Indexing,
			Sortable: f.Sortable,
		}
	})
}

// SchemaField represents a search schema field for adapters.
type SchemaField struct {
	Name     string
	Type     SchemaFieldType
	Indexing bool
	Sortable bool
}
