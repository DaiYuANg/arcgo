package mapping

import (
	"fmt"
	"reflect"
	"strconv"
	"time"
)

func (c *HashCodec) decodeField(v reflect.Value, data []byte) error {
	if len(data) == 0 {
		return nil
	}

	value := string(data)

	if v.Kind() == reflect.String {
		v.SetString(value)
		return nil
	}
	if isSignedIntKind(v.Kind()) {
		return setSignedIntField(v, value)
	}
	if isUnsignedIntKind(v.Kind()) {
		return setUnsignedIntField(v, value)
	}
	if v.Kind() == reflect.Bool {
		return setBoolField(v, value)
	}
	if isFloatKind(v.Kind()) {
		return setFloatField(v, value)
	}
	if v.Kind() == reflect.Struct {
		return c.decodeStructField(v, value, data)
	}
	if v.Kind() == reflect.Ptr {
		return c.decodePointerField(v, data)
	}

	return c.unmarshalValue(data, v.Addr().Interface())
}

func setSignedIntField(v reflect.Value, value string) error {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fmt.Errorf("parse int %q: %w", value, err)
	}

	v.SetInt(parsed)
	return nil
}

func setUnsignedIntField(v reflect.Value, value string) error {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return fmt.Errorf("parse uint %q: %w", value, err)
	}

	v.SetUint(parsed)
	return nil
}

func setBoolField(v reflect.Value, value string) error {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("parse bool %q: %w", value, err)
	}

	v.SetBool(parsed)
	return nil
}

func setFloatField(v reflect.Value, value string) error {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("parse float %q: %w", value, err)
	}

	v.SetFloat(parsed)
	return nil
}

func (c *HashCodec) decodeStructField(v reflect.Value, value string, data []byte) error {
	if v.Type() != timeType {
		return c.unmarshalValue(data, v.Addr().Interface())
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return fmt.Errorf("parse time %q: %w", value, err)
	}

	v.Set(reflect.ValueOf(parsed))
	return nil
}

func (c *HashCodec) decodePointerField(v reflect.Value, data []byte) error {
	if v.IsNil() {
		v.Set(reflect.New(v.Type().Elem()))
	}

	return c.unmarshalValue(data, v.Interface())
}
