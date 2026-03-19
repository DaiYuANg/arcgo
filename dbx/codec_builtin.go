package dbx

import (
	"database/sql"
	"encoding"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func registerBuiltinCodecs(registry *codecRegistry) {
	if registry == nil {
		return
	}
	registry.mustRegister(jsonCodec{})
	registry.mustRegister(textCodec{})
	registry.mustRegister(newTimeStringCodec("rfc3339_time", time.RFC3339))
	registry.mustRegister(newTimeStringCodec("rfc3339nano_time", time.RFC3339Nano))
	registry.mustRegister(newUnixTimeCodec("unix_time", unixSeconds))
	registry.mustRegister(newUnixTimeCodec("unix_milli_time", unixMillis))
	registry.mustRegister(newUnixTimeCodec("unix_nano_time", unixNanos))
}

type textCodec struct{}
type timeStringCodec struct {
	name   string
	layout string
}

type unixTimeCodec struct {
	name string
	unit unixUnit
}

type unixUnit int

const (
	unixSeconds unixUnit = iota
	unixMillis
	unixNanos
)

func newTimeStringCodec(name, layout string) Codec {
	return timeStringCodec{name: name, layout: layout}
}

func newUnixTimeCodec(name string, unit unixUnit) Codec {
	return unixTimeCodec{name: name, unit: unit}
}

func (textCodec) Name() string {
	return "text"
}

func (textCodec) Decode(src any, target reflect.Value) error {
	if src == nil {
		resetFieldValue(target)
		return nil
	}

	text, err := normalizeStringSource(src)
	if err != nil {
		return fmt.Errorf("dbx: codec %q: %w", "text", err)
	}
	switch {
	case target.Kind() == reflect.String:
		target.SetString(text)
		return nil
	case target.Kind() == reflect.Slice && target.Type().Elem().Kind() == reflect.Uint8:
		target.SetBytes([]byte(text))
		return nil
	}

	unmarshaler, err := resolveTextUnmarshaler(target)
	if err != nil {
		return err
	}
	if unmarshaler == nil {
		return fmt.Errorf("dbx: codec %q cannot decode into %s", "text", target.Type())
	}
	return unmarshaler.UnmarshalText([]byte(text))
}

func (textCodec) Encode(source reflect.Value) (any, error) {
	if !source.IsValid() || isNilValue(source) {
		return nil, nil
	}

	switch {
	case source.Kind() == reflect.String:
		return source.String(), nil
	case source.Kind() == reflect.Slice && source.Type().Elem().Kind() == reflect.Uint8:
		return append([]byte(nil), source.Bytes()...), nil
	}

	if marshaler := resolveTextMarshaler(source); marshaler != nil {
		text, err := marshaler.MarshalText()
		if err != nil {
			return nil, err
		}
		return string(text), nil
	}
	if stringer := resolveStringer(source); stringer != nil {
		return stringer.String(), nil
	}
	return nil, fmt.Errorf("dbx: codec %q cannot encode %s", "text", source.Type())
}

func (c timeStringCodec) Name() string {
	return c.name
}

func (c timeStringCodec) Decode(src any, target reflect.Value) error {
	if src == nil {
		resetFieldValue(target)
		return nil
	}

	text, err := normalizeStringSource(src)
	if err != nil {
		return fmt.Errorf("dbx: codec %q: %w", c.name, err)
	}
	if strings.TrimSpace(text) == "" {
		resetFieldValue(target)
		return nil
	}

	parsed, err := time.Parse(c.layout, text)
	if err != nil {
		return err
	}
	return assignDecodedValue(target, reflect.ValueOf(parsed))
}

func (c timeStringCodec) Encode(source reflect.Value) (any, error) {
	if !source.IsValid() || isNilValue(source) {
		return nil, nil
	}
	value, ok := codecValueAs[time.Time](source)
	if !ok {
		return nil, fmt.Errorf("dbx: codec %q cannot encode %s as time.Time", c.name, source.Type())
	}
	return value.Format(c.layout), nil
}

func (c unixTimeCodec) Name() string {
	return c.name
}

func (c unixTimeCodec) Decode(src any, target reflect.Value) error {
	if src == nil {
		resetFieldValue(target)
		return nil
	}

	value, err := normalizeInt64Source(src)
	if err != nil {
		return fmt.Errorf("dbx: codec %q: %w", c.name, err)
	}
	return assignDecodedValue(target, reflect.ValueOf(c.timeFromValue(value)))
}

func (c unixTimeCodec) Encode(source reflect.Value) (any, error) {
	if !source.IsValid() || isNilValue(source) {
		return nil, nil
	}
	value, ok := codecValueAs[time.Time](source)
	if !ok {
		return nil, fmt.Errorf("dbx: codec %q cannot encode %s as time.Time", c.name, source.Type())
	}
	return c.valueFromTime(value), nil
}

func (c unixTimeCodec) timeFromValue(value int64) time.Time {
	switch c.unit {
	case unixMillis:
		return time.UnixMilli(value)
	case unixNanos:
		return time.Unix(0, value)
	default:
		return time.Unix(value, 0)
	}
}

func (c unixTimeCodec) valueFromTime(value time.Time) int64 {
	switch c.unit {
	case unixMillis:
		return value.UnixMilli()
	case unixNanos:
		return value.UnixNano()
	default:
		return value.Unix()
	}
}

func normalizeStringSource(src any) (string, error) {
	switch value := src.(type) {
	case string:
		return value, nil
	case []byte:
		return string(value), nil
	case sql.RawBytes:
		return string(value), nil
	default:
		return "", fmt.Errorf("unsupported string codec source %T", src)
	}
}

func normalizeInt64Source(src any) (int64, error) {
	switch value := src.(type) {
	case int64:
		return value, nil
	case int:
		return int64(value), nil
	case int32:
		return int64(value), nil
	case int16:
		return int64(value), nil
	case int8:
		return int64(value), nil
	case uint64:
		return int64(value), nil
	case uint32:
		return int64(value), nil
	case uint16:
		return int64(value), nil
	case uint8:
		return int64(value), nil
	case []byte:
		return parseInt64(string(value))
	case sql.RawBytes:
		return parseInt64(string(value))
	case string:
		return parseInt64(value)
	default:
		return 0, fmt.Errorf("unsupported unix time codec source %T", src)
	}
}

func parseInt64(input string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(input), 10, 64)
}

func resolveTextUnmarshaler(target reflect.Value) (encoding.TextUnmarshaler, error) {
	if !target.CanSet() {
		return nil, fmt.Errorf("dbx: codec target is not settable")
	}
	if target.Kind() == reflect.Pointer {
		if target.IsNil() {
			target.Set(reflect.New(target.Type().Elem()))
		}
		if unmarshaler, ok := target.Interface().(encoding.TextUnmarshaler); ok {
			return unmarshaler, nil
		}
	}
	if target.CanAddr() {
		if unmarshaler, ok := target.Addr().Interface().(encoding.TextUnmarshaler); ok {
			return unmarshaler, nil
		}
	}
	return nil, nil
}

func resolveTextMarshaler(source reflect.Value) encoding.TextMarshaler {
	if !source.IsValid() {
		return nil
	}
	if marshaler, ok := source.Interface().(encoding.TextMarshaler); ok {
		return marshaler
	}
	if source.Kind() != reflect.Pointer && source.CanAddr() {
		if marshaler, ok := source.Addr().Interface().(encoding.TextMarshaler); ok {
			return marshaler
		}
	}
	return nil
}

func resolveStringer(source reflect.Value) fmt.Stringer {
	if !source.IsValid() {
		return nil
	}
	if stringer, ok := source.Interface().(fmt.Stringer); ok {
		return stringer
	}
	if source.Kind() != reflect.Pointer && source.CanAddr() {
		if stringer, ok := source.Addr().Interface().(fmt.Stringer); ok {
			return stringer
		}
	}
	return nil
}
