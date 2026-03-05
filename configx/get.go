package configx

import "errors"

var errNilConfig = errors.New("config is nil")

// GetAs 读取并转换指定路径配置为目标类型。
func GetAs[T any](cfg *Config, path string) (T, error) {
	var zero T
	if cfg == nil {
		return zero, errNilConfig
	}

	var out T
	if err := cfg.Unmarshal(path, &out); err != nil {
		return zero, err
	}
	return out, nil
}

// GetAsOr 读取并转换指定路径配置，失败时返回回退值。
func GetAsOr[T any](cfg *Config, path string, fallback T) T {
	if cfg == nil {
		return fallback
	}
	if path != "" && !cfg.Exists(path) {
		return fallback
	}

	out, err := GetAs[T](cfg, path)
	if err != nil {
		return fallback
	}
	return out
}

// MustGetAs 读取并转换指定路径配置，失败时 panic。
func MustGetAs[T any](cfg *Config, path string) T {
	out, err := GetAs[T](cfg, path)
	if err != nil {
		panic(err)
	}
	return out
}
