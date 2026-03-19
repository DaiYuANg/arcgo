package dbx

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/DaiYuANg/arcgo/collectionx"
)

type Codec interface {
	Name() string
	Decode(src any, target reflect.Value) error
	Encode(source reflect.Value) (any, error)
}

type codecRegistry struct {
	codecs collectionx.ConcurrentMap[string, Codec]
}

type typedCodec[T any] struct {
	name   string
	decode func(any) (T, error)
	encode func(T) (any, error)
}

type jsonCodec struct{}

func NewCodec[T any](name string, decode func(any) (T, error), encode func(T) (any, error)) Codec {
	return typedCodec[T]{
		name:   normalizeCodecName(name),
		decode: decode,
		encode: encode,
	}
}

func RegisterCodec(codec Codec) error {
	return defaultMapperRuntime.codecs.register(codec)
}

func MustRegisterCodec(codec Codec) {
	if err := RegisterCodec(codec); err != nil {
		panic(err)
	}
}

func LookupCodec(name string) (Codec, bool) {
	return defaultMapperRuntime.codecs.get(name)
}

func newCodecRegistry() *codecRegistry {
	return &codecRegistry{
		codecs: collectionx.NewConcurrentMapWithCapacity[string, Codec](4),
	}
}

func (r *codecRegistry) clone() *codecRegistry {
	if r == nil {
		return newCodecRegistry()
	}
	cloned := newCodecRegistry()
	cloned.codecs.SetAll(r.codecs.All())
	return cloned
}

func (r *codecRegistry) register(codec Codec) error {
	if isNilCodec(codec) {
		return ErrNilCodec
	}

	name := normalizeCodecName(codec.Name())
	if name == "" {
		return fmt.Errorf("dbx: codec name cannot be empty")
	}

	if _, loaded := r.codecs.GetOrStore(name, codec); loaded {
		return fmt.Errorf("dbx: codec %q is already registered", name)
	}
	return nil
}

func (r *codecRegistry) mustRegister(codec Codec) {
	if err := r.register(codec); err != nil {
		panic(err)
	}
}

func (r *codecRegistry) get(name string) (Codec, bool) {
	return r.codecs.Get(normalizeCodecName(name))
}

func (c typedCodec[T]) Name() string {
	return c.name
}

func (c typedCodec[T]) Decode(src any, target reflect.Value) error {
	if src == nil {
		resetFieldValue(target)
		return nil
	}

	value, err := c.decode(src)
	if err != nil {
		return err
	}
	return assignDecodedValue(target, reflect.ValueOf(value))
}

func (c typedCodec[T]) Encode(source reflect.Value) (any, error) {
	if !source.IsValid() || isNilValue(source) {
		return nil, nil
	}

	value, ok := codecValueAs[T](source)
	if !ok {
		return nil, fmt.Errorf("dbx: codec %q cannot encode %s as %s", c.name, source.Type(), reflect.TypeFor[T]())
	}
	return c.encode(value)
}

func (jsonCodec) Name() string {
	return "json"
}

func (jsonCodec) Decode(src any, target reflect.Value) error {
	if src == nil {
		resetFieldValue(target)
		return nil
	}

	payload, err := normalizeJSONPayload(src)
	if err != nil {
		return err
	}
	if len(payload) == 0 {
		resetFieldValue(target)
		return nil
	}

	destination, err := codecDecodeTarget(target)
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, destination.Interface())
}

func (jsonCodec) Encode(source reflect.Value) (any, error) {
	if !source.IsValid() || isNilValue(source) {
		return nil, nil
	}
	payload, err := json.Marshal(source.Interface())
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func normalizeCodecName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func isNilCodec(codec Codec) bool {
	if codec == nil {
		return true
	}
	value := reflect.ValueOf(codec)
	return value.Kind() == reflect.Pointer && value.IsNil()
}

func codecValueAs[T any](source reflect.Value) (T, bool) {
	var zero T
	if !source.IsValid() {
		return zero, false
	}
	if value, ok := source.Interface().(T); ok {
		return value, true
	}
	if source.Kind() == reflect.Pointer && !source.IsNil() {
		if value, ok := source.Elem().Interface().(T); ok {
			return value, true
		}
	}
	return zero, false
}

func assignDecodedValue(target reflect.Value, value reflect.Value) error {
	if !target.CanSet() {
		return fmt.Errorf("dbx: codec target is not settable")
	}
	if !value.IsValid() {
		resetFieldValue(target)
		return nil
	}

	if target.Kind() == reflect.Pointer {
		if value.Type().AssignableTo(target.Type()) {
			target.Set(value)
			return nil
		}
		if value.Kind() == reflect.Pointer {
			if value.IsNil() {
				resetFieldValue(target)
				return nil
			}
			value = value.Elem()
		}
		holder := reflect.New(target.Type().Elem())
		if err := assignDecodedValue(holder.Elem(), value); err != nil {
			return err
		}
		target.Set(holder)
		return nil
	}

	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			resetFieldValue(target)
			return nil
		}
		value = value.Elem()
	}
	if value.Type().AssignableTo(target.Type()) {
		target.Set(value)
		return nil
	}
	if value.Type().ConvertibleTo(target.Type()) {
		target.Set(value.Convert(target.Type()))
		return nil
	}
	return fmt.Errorf("dbx: decoded codec value %s cannot be assigned to %s", value.Type(), target.Type())
}

func codecDecodeTarget(target reflect.Value) (reflect.Value, error) {
	if !target.CanSet() {
		return reflect.Value{}, fmt.Errorf("dbx: codec target is not settable")
	}
	if target.Kind() == reflect.Pointer {
		if target.IsNil() {
			target.Set(reflect.New(target.Type().Elem()))
		}
		return target, nil
	}
	return target.Addr(), nil
}

func resetFieldValue(target reflect.Value) {
	if target.CanSet() {
		target.Set(reflect.Zero(target.Type()))
	}
}

func isNilValue(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface:
		return value.IsNil()
	default:
		return false
	}
}

func normalizeJSONPayload(src any) ([]byte, error) {
	switch value := src.(type) {
	case []byte:
		return append([]byte(nil), value...), nil
	case sql.RawBytes:
		return append([]byte(nil), value...), nil
	case string:
		return []byte(value), nil
	default:
		return nil, fmt.Errorf("dbx: json codec does not support source type %T", src)
	}
}
