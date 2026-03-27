package mapping

import (
	"fmt"
	"reflect"
	"strconv"
	"time"
)

func (c *HashCodec) encodeField(v reflect.Value) ([]byte, error) {
	if !v.IsValid() {
		return []byte(""), nil
	}

	if v.Kind() == reflect.String {
		return []byte(v.String()), nil
	}
	if isSignedIntKind(v.Kind()) {
		return []byte(strconv.FormatInt(v.Int(), 10)), nil
	}
	if isUnsignedIntKind(v.Kind()) {
		return []byte(strconv.FormatUint(v.Uint(), 10)), nil
	}
	if v.Kind() == reflect.Bool {
		return encodeBoolField(v.Bool()), nil
	}
	if isFloatKind(v.Kind()) {
		return []byte(strconv.FormatFloat(v.Float(), 'f', -1, 64)), nil
	}
	if v.Kind() == reflect.Struct {
		return c.encodeStructField(v)
	}

	return c.marshalValue(v.Interface())
}

func encodeBoolField(value bool) []byte {
	if value {
		return []byte("1")
	}
	return []byte("0")
}

func (c *HashCodec) encodeStructField(v reflect.Value) ([]byte, error) {
	if v.Type() != timeType {
		return c.marshalValue(v.Interface())
	}

	timeValue, ok := v.Interface().(time.Time)
	if !ok {
		return nil, fmt.Errorf("expected %s, got %T", timeType, v.Interface())
	}

	return []byte(timeValue.Format(time.RFC3339)), nil
}
