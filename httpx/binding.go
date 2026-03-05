package httpx

import (
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/go-chi/chi/v5"
)

var (
	durationType       = reflect.TypeOf(time.Duration(0))
	timeType           = reflect.TypeOf(time.Time{})
	textUnmarshalIface = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
)

func bindRequestToInput[T any](r *http.Request, in *T) error {
	if r == nil || in == nil {
		return nil
	}

	if custom, ok := any(in).(RequestBinder); ok {
		return custom.BindRequest(r)
	}

	if err := bindRequestBody(r, in); err != nil {
		return err
	}

	if err := bindRequestParams(r, in); err != nil {
		return err
	}

	return nil
}

func bindRequestBody[T any](r *http.Request, in *T) error {
	if custom, ok := any(in).(RequestBodyBinder); ok {
		return custom.BindRequestBody(r)
	}
	return decodeJSONBody(r, in)
}

func bindRequestParams[T any](r *http.Request, in *T) error {
	if custom, ok := any(in).(RequestParamsBinder); ok {
		return custom.BindRequestParams(r)
	}

	root := reflect.ValueOf(in)
	if root.Kind() != reflect.Ptr || root.IsNil() {
		return nil
	}

	return bindStructFromRequest(r, root.Elem())
}

func decodeJSONBody[T any](r *http.Request, in *T) error {
	if !shouldDecodeBody(r) {
		return nil
	}

	target := reflect.ValueOf(in)
	if target.Kind() != reflect.Ptr || target.IsNil() {
		return nil
	}

	targetElem := target.Elem()
	if targetElem.Kind() == reflect.Struct {
		if bodyField := targetElem.FieldByName("Body"); bodyField.IsValid() && bodyField.CanSet() {
			return decodeJSONIntoValue(r.Body, bodyField)
		}
	}

	return decodeJSONIntoValue(r.Body, targetElem)
}

func decodeJSONIntoValue(body io.ReadCloser, value reflect.Value) error {
	if body == nil || body == http.NoBody {
		return nil
	}

	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()

	target := value
	if target.Kind() != reflect.Ptr {
		if !target.CanAddr() {
			return nil
		}
		target = target.Addr()
	}

	if err := decoder.Decode(target.Interface()); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return fmt.Errorf("decode request body: %w", err)
	}

	return nil
}

func bindStructFromRequest(r *http.Request, v reflect.Value) error {
	if !v.IsValid() {
		return nil
	}

	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		if field.Anonymous && fieldValue.Kind() == reflect.Struct {
			if err := bindStructFromRequest(r, fieldValue); err != nil {
				return err
			}
		}

		if value, ok := readParamValue(r, field); ok {
			if err := setFieldValue(fieldValue, value); err != nil {
				return fmt.Errorf("bind field %s: %w", field.Name, err)
			}
		}
	}

	return nil
}

func readParamValue(r *http.Request, field reflect.StructField) (string, bool) {
	if name, ok := field.Tag.Lookup("query"); ok {
		key := fallbackTagName(name, field.Name)
		raw := r.URL.Query().Get(key)
		if raw != "" {
			return raw, true
		}
	}

	if name, ok := field.Tag.Lookup("header"); ok {
		key := fallbackTagName(name, field.Name)
		raw := r.Header.Get(key)
		if raw != "" {
			return raw, true
		}
	}

	if name, ok := field.Tag.Lookup("cookie"); ok {
		key := fallbackTagName(name, field.Name)
		if c, err := r.Cookie(key); err == nil && c != nil && c.Value != "" {
			return c.Value, true
		}
	}

	if name, ok := field.Tag.Lookup("path"); ok {
		key := fallbackTagName(name, field.Name)
		if raw := readPathParam(r, key); raw != "" {
			return raw, true
		}
	}

	return "", false
}

func readPathParam(r *http.Request, name string) string {
	if r == nil || name == "" {
		return ""
	}

	if v := adapter.RouteParam(r.Context(), name); v != "" {
		return v
	}

	if v := r.PathValue(name); v != "" {
		return v
	}

	return chi.URLParam(r, name)
}

func fallbackTagName(tagValue, fieldName string) string {
	name := strings.TrimSpace(tagValue)
	if name != "" {
		return name
	}
	return strings.ToLower(fieldName)
}

func setFieldValue(v reflect.Value, raw string) error {
	if !v.CanSet() {
		return nil
	}

	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return setFieldValue(v.Elem(), raw)
	}

	if v.CanAddr() {
		addr := v.Addr()
		if addr.Type().Implements(textUnmarshalIface) {
			return addr.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(raw))
		}
	}

	if v.Type() == durationType {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return err
		}
		v.SetInt(int64(d))
		return nil
	}

	if v.Type() == timeType {
		tm, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(tm))
		return nil
	}

	switch v.Kind() {
	case reflect.String:
		v.SetString(raw)
	case reflect.Bool:
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		v.SetBool(parsed)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(raw, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetInt(parsed)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		parsed, err := strconv.ParseUint(raw, 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetUint(parsed)
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(raw, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetFloat(parsed)
	case reflect.Slice:
		return setSliceValue(v, raw)
	default:
		return fmt.Errorf("unsupported field type %s", v.Type())
	}

	return nil
}

func setSliceValue(v reflect.Value, raw string) error {
	items := strings.Split(raw, ",")
	elemType := v.Type().Elem()
	slice := reflect.MakeSlice(v.Type(), 0, len(items))

	for _, item := range items {
		item = strings.TrimSpace(item)
		elem := reflect.New(elemType).Elem()
		if err := setFieldValue(elem, item); err != nil {
			return err
		}
		slice = reflect.Append(slice, elem)
	}

	v.Set(slice)
	return nil
}
