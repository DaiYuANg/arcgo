package configx

import (
	"reflect"
	"strings"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
)

// loadDefaultsStruct 将结构体转换为 map 并加载到 koanf
func loadDefaultsStruct(k *koanf.Koanf, defaults any) error {
	defaultMap, err := structToMap(defaults)
	if err != nil {
		return err
	}
	return k.Load(confmap.Provider(defaultMap, "."), nil)
}

// structToMap 使用 reflect 将 struct 转换为 map[string]any
// 支持 map[string]any, map[string]interface{}, struct
func structToMap(s any) (map[string]any, error) {
	// Case 1: already map[string]any
	if m, ok := s.(map[string]any); ok {
		return m, nil
	}

	if s == nil {
		return nil, &structToMapError{"expected struct or map, got <nil>"}
	}

	result := make(map[string]any)

	// Case 2: struct
	v := reflect.ValueOf(s)
	t := reflect.TypeOf(s)

	// 如果是指针，解引用
	if t.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil, &structToMapError{"expected struct or map, got " + t.Kind().String()}
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		// 跳过未导出字段
		if !value.CanInterface() {
			continue
		}

		// 获取 mapstructure 标签， fallback 到字段名
		tag := field.Tag.Get("mapstructure")
		if tag == "" {
			tag = strings.ToLower(field.Name)
		} else if tag == "-" {
			continue // 忽略
		}

		// 处理嵌套 struct（递归）
		if value.Kind() == reflect.Struct {
			nested, err := structToMap(value.Interface())
			if err != nil {
				return nil, err
			}
			result[tag] = nested
		} else {
			result[tag] = value.Interface()
		}
	}

	return result, nil
}

type structToMapError struct {
	msg string
}

func (e *structToMapError) Error() string { return e.msg }
