package mapping

import (
	"fmt"
	"reflect"
	"strconv"
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
func (b *KeyBuilder) Build(entity any, metadata *EntityMetadata) (string, error) {
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
	return b.prefix + ":" + id
}

// BuildIndexKey builds an index key for a given field.
func (b *KeyBuilder) BuildIndexKey(fieldName string) string {
	if b.prefix == "" {
		return "idx:" + fieldName
	}
	return b.prefix + ":idx:" + fieldName
}

// BuildFieldKey builds a key for a secondary index field.
func (b *KeyBuilder) BuildFieldKey(fieldName, fieldValue string) string {
	return b.BuildIndexKey(fieldName) + ":" + fieldValue + ":" + fieldName
}

func (b *KeyBuilder) formatValue(v reflect.Value) string {
	if v.Kind() == reflect.String {
		return v.String()
	}
	if isSignedIntKind(v.Kind()) {
		return strconv.FormatInt(v.Int(), 10)
	}
	if isUnsignedIntKind(v.Kind()) {
		return strconv.FormatUint(v.Uint(), 10)
	}
	return fmt.Sprint(v.Interface())
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
