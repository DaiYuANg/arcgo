package mapping

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/samber/mo"
)

// Serializer defines the interface for serializing/deserializing values.
type Serializer interface {
	// Marshal serializes a value to bytes.
	Marshal(v interface{}) ([]byte, error)
	// Unmarshal deserializes bytes to a value.
	Unmarshal(data []byte, v interface{}) error
}

// JSONSerializer implements Serializer using encoding/json.
type JSONSerializer struct{}

// NewJSONSerializer creates a new JSONSerializer.
func NewJSONSerializer() *JSONSerializer {
	return &JSONSerializer{}
}

// Marshal implements Serializer.Marshal.
func (s *JSONSerializer) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal implements Serializer.Unmarshal.
func (s *JSONSerializer) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// HashCodec encodes/decodes struct fields to/from hash field-value pairs.
type HashCodec struct {
	serializer Serializer
}

// NewHashCodec creates a new HashCodec.
func NewHashCodec(serializer Serializer) *HashCodec {
	return &HashCodec{
		serializer: mo.TupleToOption(serializer, serializer != nil).OrElse(NewJSONSerializer()),
	}
}

// Encode encodes an entity to a hash map.
func (c *HashCodec) Encode(entity interface{}, metadata *EntityMetadata) (map[string][]byte, error) {
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	result := make(map[string][]byte, len(metadata.Fields))

	for fieldName, fieldTag := range metadata.Fields {
		fieldVal := v.FieldByName(fieldName)
		if !fieldVal.IsValid() {
			continue
		}

		data, err := c.encodeField(fieldVal)
		if err != nil {
			return nil, fmt.Errorf("encode field %s: %w", fieldName, err)
		}

		result[storageFieldName(fieldName, fieldTag)] = data
	}

	return result, nil
}

// Decode decodes a hash map to an entity.
func (c *HashCodec) Decode(data map[string][]byte, entity interface{}, metadata *EntityMetadata) error {
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Build reverse map: store name -> field name
	storeToField := make(map[string]string)
	for fieldName, fieldTag := range metadata.Fields {
		storeToField[storageFieldName(fieldName, fieldTag)] = fieldName
	}

	for storeName, fieldData := range data {
		fieldName, ok := storeToField[storeName]
		if !ok {
			continue
		}

		field := v.FieldByName(fieldName)
		if !field.IsValid() || !field.CanSet() {
			continue
		}

		if err := c.decodeField(field, fieldData); err != nil {
			return fmt.Errorf("decode field %s: %w", fieldName, err)
		}
	}

	return nil
}

func (c *HashCodec) encodeField(v reflect.Value) ([]byte, error) {
	if !v.IsValid() {
		return []byte(""), nil
	}

	switch v.Kind() {
	case reflect.String:
		return []byte(v.String()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return []byte(strconv.FormatInt(v.Int(), 10)), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return []byte(strconv.FormatUint(v.Uint(), 10)), nil
	case reflect.Bool:
		if v.Bool() {
			return []byte("1"), nil
		}
		return []byte("0"), nil
	case reflect.Float32, reflect.Float64:
		return []byte(strconv.FormatFloat(v.Float(), 'f', -1, 64)), nil
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(time.Time{}) {
			t := v.Interface().(time.Time)
			return []byte(t.Format(time.RFC3339)), nil
		}
		// Fall back to JSON for other structs
		return c.serializer.Marshal(v.Interface())
	case reflect.Slice, reflect.Array, reflect.Map, reflect.Ptr, reflect.Interface:
		return c.serializer.Marshal(v.Interface())
	default:
		return c.serializer.Marshal(v.Interface())
	}
}

func (c *HashCodec) decodeField(v reflect.Value, data []byte) error {
	if len(data) == 0 {
		return nil
	}

	str := string(data)

	switch v.Kind() {
	case reflect.String:
		v.SetString(str)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return err
		}
		v.SetInt(i)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, err := strconv.ParseUint(str, 10, 64)
		if err != nil {
			return err
		}
		v.SetUint(i)
		return nil
	case reflect.Bool:
		v.SetBool(str == "1" || str == "true")
		return nil
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return err
		}
		v.SetFloat(f)
		return nil
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(time.Time{}) {
			t, err := time.Parse(time.RFC3339, str)
			if err != nil {
				return err
			}
			v.Set(reflect.ValueOf(t))
			return nil
		}
		// Fall back to JSON for other structs
		return c.serializer.Unmarshal(data, v.Addr().Interface())
	case reflect.Ptr:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return c.serializer.Unmarshal(data, v.Interface())
	default:
		return c.serializer.Unmarshal(data, v.Addr().Interface())
	}
}

// EncodeSingleValue encodes a single value to bytes.
func (c *HashCodec) EncodeSingleValue(value interface{}) ([]byte, error) {
	v := reflect.ValueOf(value)
	return c.encodeField(v)
}

func storageFieldName(fieldName string, fieldTag FieldTag) string {
	if fieldTag.Name != "" {
		return fieldTag.Name
	}
	return fieldName
}
