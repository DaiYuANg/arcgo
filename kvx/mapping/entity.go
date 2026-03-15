package mapping

import (
	"reflect"
	"strings"
	"sync"
)

// FieldTag represents metadata for a struct field.
type FieldTag struct {
	Name      string // field name in storage
	Ignored   bool   // whether to ignore this field
	Index     bool   // whether this field is indexed
	IndexName string // custom index name
}

// EntityMetadata holds metadata for an entity type.
type EntityMetadata struct {
	Type            reflect.Type
	KeyField        string // field name for the entity key/ID
	KeyPrefix       string // prefix for generating keys
	Fields          map[string]FieldTag
	IndexFields     []string // list of indexed field names
	HasExpiration   bool
	ExpirationField string
}

// TagParser parses struct tags into metadata.
type TagParser struct {
	cache sync.Map // map[reflect.Type]*EntityMetadata
}

// NewTagParser creates a new TagParser.
func NewTagParser() *TagParser {
	return &TagParser{}
}

// Parse parses metadata from a struct type.
func (p *TagParser) Parse(t reflect.Type) (*EntityMetadata, error) {
	// Check cache first
	if cached, ok := p.cache.Load(t); ok {
		return cached.(*EntityMetadata), nil
	}

	return p.ParseType(reflect.Zero(t).Interface())
}

// ParseType parses metadata from an entity instance.
func (p *TagParser) ParseType(entity interface{}) (*EntityMetadata, error) {
	t := reflect.TypeOf(entity)

	// Check cache first
	if cached, ok := p.cache.Load(t); ok {
		return cached.(*EntityMetadata), nil
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, ErrNonStructType
	}

	metadata := &EntityMetadata{
		Type:        t,
		Fields:      make(map[string]FieldTag),
		IndexFields: make([]string, 0),
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		tag := field.Tag.Get("kvx")
		if tag == "" {
			continue
		}

		fieldTag := p.parseFieldTag(tag)

		if fieldTag.Ignored {
			continue
		}

		// Check for key field
		if fieldTag.Name == "id" || fieldTag.Name == "key" {
			metadata.KeyField = field.Name
			continue
		}

		metadata.Fields[field.Name] = fieldTag

		if fieldTag.Index {
			metadata.IndexFields = append(metadata.IndexFields, field.Name)
		}
	}

	p.cache.Store(t, metadata)
	return metadata, nil
}

func (p *TagParser) parseFieldTag(tag string) FieldTag {
	result := FieldTag{}
	parts := strings.Split(tag, ",")

	if len(parts) > 0 {
		name := strings.TrimSpace(parts[0])
		switch name {
		case "-":
			result.Ignored = true
		case "id", "key":
			result.Name = name
		default:
			result.Name = name
		}
	}

	for _, part := range parts[1:] {
		part = strings.TrimSpace(part)
		if part == "omitempty" {
			continue
		} else if part == "index" {
			result.Index = true
		} else if strings.HasPrefix(part, "index=") {
			result.Index = true
			result.IndexName = strings.TrimPrefix(part, "index=")
		} else if part == "ignore" {
			result.Ignored = true
		}
	}

	return result
}

// GetCached returns cached metadata for a type if available.
func (p *TagParser) GetCached(t reflect.Type) *EntityMetadata {
	if cached, ok := p.cache.Load(t); ok {
		return cached.(*EntityMetadata)
	}
	return nil
}

// Errors
var (
	ErrNonStructType = &parseError{"non-struct type"}
)

type parseError struct {
	msg string
}

func (e *parseError) Error() string {
	return "kvx: " + e.msg
}
