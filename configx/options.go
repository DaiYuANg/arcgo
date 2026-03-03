package configx

import (
	"github.com/go-playground/validator/v10"
	"github.com/samber/mo"
)

// Source 配置来源
type Source int

const (
	SourceDotenv Source = iota
	SourceFile
	SourceEnv
)

// ValidateLevel 验证级别
type ValidateLevel int

const (
	ValidateLevelNone ValidateLevel = iota
	ValidateLevelStruct
	ValidateLevelRequired
)

// Options 配置加载选项
type Options struct {
	dotenvFiles     []string
	envPrefix       string
	files           []string
	priority        []Source
	defaults        mo.Option[map[string]any]
	defaultsStruct  any
	validate        *validator.Validate
	validateLevel   ValidateLevel
	ignoreDotenvErr bool
}

// Option 配置选项函数（非泛型）
type Option func(*Options)

// NewOptions 创建默认选项
func NewOptions() *Options {
	return &Options{
		dotenvFiles:     []string{".env", ".env.local"},
		priority:        []Source{SourceDotenv, SourceFile, SourceEnv},
		validateLevel:   ValidateLevelNone,
		ignoreDotenvErr: true,
	}
}

// WithDotenv 启用 .env 文件加载
func WithDotenv(files ...string) Option {
	return func(o *Options) {
		if len(files) > 0 {
			o.dotenvFiles = files
		}
	}
}

// WithEnvPrefix 设置环境变量前缀
func WithEnvPrefix(prefix string) Option {
	return func(o *Options) { o.envPrefix = prefix }
}

// WithFiles 设置配置文件路径
func WithFiles(files ...string) Option {
	return func(o *Options) { o.files = files }
}

// WithPriority 设置配置源优先级
func WithPriority(p ...Source) Option {
	return func(o *Options) { o.priority = p }
}

// WithDefaults 设置默认值（map 形式）
func WithDefaults(m map[string]any) Option {
	return func(o *Options) {
		o.defaults = mo.Some(m)
	}
}

// WithDefaultsStruct 设置默认值（struct 形式）
func WithDefaultsStruct(s any) Option {
	return func(o *Options) {
		o.defaultsStruct = s
	}
}

// WithValidator 设置自定义 validator
func WithValidator(v *validator.Validate) Option {
	return func(o *Options) { o.validate = v }
}

// WithValidateLevel 设置验证级别
func WithValidateLevel(level ValidateLevel) Option {
	return func(o *Options) { o.validateLevel = level }
}

// WithIgnoreDotenvError 设置是否忽略 .env 加载错误
func WithIgnoreDotenvError(ignore bool) Option {
	return func(o *Options) { o.ignoreDotenvErr = ignore }
}
