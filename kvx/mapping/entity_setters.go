package mapping

import (
	"fmt"
	"reflect"
	"time"
)

func setFieldStringValue(field reflect.Value, value string) error {
	if field.Type() == timeType {
		return setTimeField(field, value)
	}
	if field.Kind() == reflect.String {
		return setStringField(field, value)
	}
	if isSignedIntKind(field.Kind()) {
		return setSignedIntField(field, value)
	}
	if isUnsignedIntKind(field.Kind()) {
		return setUnsignedIntField(field, value)
	}
	if setter := directFieldSetter(field.Kind()); setter != nil {
		return setter(field, value)
	}

	return nil
}

func directFieldSetter(kind reflect.Kind) func(reflect.Value, string) error {
	if kind == reflect.String {
		return setStringField
	}
	if kind == reflect.Bool {
		return setBoolField
	}
	if isFloatKind(kind) {
		return setFloatField
	}

	return nil
}

func setStringField(field reflect.Value, value string) error {
	field.SetString(value)
	return nil
}

func setTimeField(field reflect.Value, value string) error {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return fmt.Errorf("parse time %q: %w", value, err)
	}

	field.Set(reflect.ValueOf(parsed))
	return nil
}

// Errors
var (
	ErrNonStructType     = &parseError{"non-struct type"}
	ErrNonPointerValue   = &parseError{"non-pointer value"}
	ErrNoKeyFieldDefined = &parseError{"no key field defined"}
)

type parseError struct {
	msg string
}

func (e *parseError) Error() string {
	return "kvx: " + e.msg
}
