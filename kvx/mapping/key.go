package mapping

import (
	"fmt"
	"reflect"
	"strings"
)

// KeyBuilder builds Redis keys from entity metadata.
type KeyBuilder struct {
	prefix string
}

// NewKeyBuilder creates a new KeyBuilder with the given prefix.
func NewKeyBuilder(prefix string) *KeyBuilder {
	return &KeyBuilder{prefix: strings.TrimSuffix(prefix, ":")}
}

// Build builds a key from an entity's ID field value.
func (b *KeyBuilder) Build(entity interface{}, metadata *EntityMetadata) (string, error) {
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if metadata.KeyField == "" {
		return "", ErrNoKeyField
	}

	keyFieldValue := v.FieldByName(metadata.KeyField)
	if !keyFieldValue.IsValid() {
		return "", ErrKeyFieldNotFound
	}

	id := b.formatValue(keyFieldValue)
	if id == "" {
		return "", ErrEmptyKeyValue
	}

	return b.BuildWithID(id), nil
}

// BuildWithID builds a key from a raw ID value.
func (b *KeyBuilder) BuildWithID(id string) string {
	if b.prefix == "" {
		return id
	}
	return fmt.Sprintf("%s:%s", b.prefix, id)
}

// BuildIndexKey builds an index key for a given field.
func (b *KeyBuilder) BuildIndexKey(fieldName string) string {
	if b.prefix == "" {
		return fmt.Sprintf("idx:%s", fieldName)
	}
	return fmt.Sprintf("%s:idx:%s", b.prefix, fieldName)
}

// BuildFieldKey builds a key for a secondary index field.
func (b *KeyBuilder) BuildFieldKey(fieldName string, fieldValue string) string {
	return fmt.Sprintf("%s:%s:%s", b.BuildIndexKey(fieldName), fieldValue, fieldName)
}

func (b *KeyBuilder) formatValue(v reflect.Value) string {
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", v.Uint())
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

// Errors
var (
	ErrNoKeyField       = &keyError{"no key field defined"}
	ErrKeyFieldNotFound = &keyError{"key field not found in struct"}
	ErrEmptyKeyValue    = &keyError{"empty key value"}
)

type keyError struct {
	msg string
}

func (e *keyError) Error() string {
	return "kvx: " + e.msg
}
