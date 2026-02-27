package configx

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf/v2"
)

// Config 配置对象
type Config struct {
	k        *koanf.Koanf
	validate *validator.Validate
	level    ValidateLevel
}

func newConfig(k *koanf.Koanf, opts *Options) *Config {
	v := opts.validate
	if v == nil {
		v = validator.New()
	}
	return &Config{
		k:        k,
		validate: v,
		level:    opts.validateLevel,
	}
}

// validateStruct 根据验证级别验证结构体
func (c *Config) validateStruct(out any) error {
	switch c.level {
	case ValidateLevelNone:
		return nil
	case ValidateLevelStruct, ValidateLevelRequired:
		return c.validate.Struct(out)
	default:
		return nil
	}
}

// Get 获取任意类型的配置值
func (c *Config) Get(path string) any {
	return c.k.Get(path)
}

// GetString 获取字符串配置值
func (c *Config) GetString(path string) string {
	return c.k.String(path)
}

// GetInt 获取整数配置值
func (c *Config) GetInt(path string) int {
	return c.k.Int(path)
}

// GetInt64 获取 64 位整数配置值
func (c *Config) GetInt64(path string) int64 {
	return c.k.Int64(path)
}

// GetFloat64 获取浮点数配置值
func (c *Config) GetFloat64(path string) float64 {
	return c.k.Float64(path)
}

// GetBool 获取布尔配置值
func (c *Config) GetBool(path string) bool {
	return c.k.Bool(path)
}

// GetDuration 获取时长配置值 (支持 "1s", "1m", "1h" 等格式)
func (c *Config) GetDuration(path string) time.Duration {
	return c.k.Duration(path)
}

// GetStringSlice 获取字符串切片配置值
func (c *Config) GetStringSlice(path string) []string {
	return c.k.Strings(path)
}

// GetIntSlice 获取整数切片配置值
func (c *Config) GetIntSlice(path string) []int {
	return c.k.Ints(path)
}

// Unmarshal 将配置解构到目标结构体
// path: 配置路径，空字符串表示整个配置
func (c *Config) Unmarshal(path string, out any) error {
	return c.k.Unmarshal(path, out)
}

// UnmarshalWithValidate 将配置解构到目标结构体并进行验证
// path: 配置路径，空字符串表示整个配置
func (c *Config) UnmarshalWithValidate(path string, out any) error {
	if err := c.k.Unmarshal(path, out); err != nil {
		return err
	}
	return c.validate.Struct(out)
}

// Exists 检查配置键是否存在
func (c *Config) Exists(path string) bool {
	return c.k.Exists(path)
}

// All 获取所有配置 (map 形式)
func (c *Config) All() map[string]any {
	return c.k.All()
}

// Validate 手动验证结构体
func (c *Config) Validate(out any) error {
	return c.validate.Struct(out)
}
