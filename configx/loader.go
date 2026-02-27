package configx

import (
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
	"github.com/samber/mo"
)

// Loader 配置加载器（非泛型）
type Loader struct {
	opts *Options
}

// Load 加载配置到目标结构体
func (l *Loader) Load(out any) error {
	cfg, err := l.loadInternal()
	if err != nil {
		return err
	}
	if err := cfg.k.Unmarshal("", out); err != nil {
		return err
	}
	return cfg.validateStruct(out)
}

// LoadConfig 加载并返回 Config 对象
func (l *Loader) LoadConfig() (*Config, error) {
	return l.loadInternal()
}

func (l *Loader) loadInternal() (*Config, error) {
	k := koanf.New(".")

	// 加载默认值（map 形式）
	if l.opts.defaults.IsPresent() {
		defaults, _ := l.opts.defaults.Get()
		if err := k.Load(confmap.Provider(defaults, "."), nil); err != nil {
			return nil, err
		}
	}

	// 加载默认值（struct 形式）
	if l.opts.defaultsStruct != nil {
		defaultMap, err := structToMap(l.opts.defaultsStruct)
		if err != nil {
			return nil, err
		}
		if err := k.Load(confmap.Provider(defaultMap, "."), nil); err != nil {
			return nil, err
		}
	}

	// 按优先级加载
	for _, src := range l.opts.priority {
		switch src {
		case SourceDotenv:
			if err := loadDotenv(k, l.opts.dotenvFiles, l.opts.ignoreDotenvErr); err != nil {
				return nil, err
			}
		case SourceFile:
			if err := loadFiles(k, l.opts.files); err != nil {
				return nil, err
			}
		case SourceEnv:
			if err := loadEnv(k, l.opts.envPrefix); err != nil {
				return nil, err
			}
		}
	}

	return newConfig(k, l.opts), nil
}

// New 创建非泛型加载器
func New(opts ...Option) *Loader {
	options := NewOptions()
	for _, opt := range opts {
		opt(options)
	}
	return &Loader{opts: options}
}

// LoaderT 泛型配置加载器
type LoaderT[T any] struct {
	opts *Options
}

// Load 加载配置到泛型结构体 T
func (l *LoaderT[T]) Load() mo.Result[T] {
	cfg, err := l.loadInternal()
	if err != nil {
		return mo.Err[T](err)
	}

	var out T
	if err := cfg.k.Unmarshal("", &out); err != nil {
		return mo.Err[T](err)
	}
	if err := cfg.validateStruct(out); err != nil {
		return mo.Err[T](err)
	}
	return mo.Ok(out)
}

// LoadConfig 加载并返回 Config 对象
func (l *LoaderT[T]) LoadConfig() (*Config, error) {
	return l.loadInternal()
}

func (l *LoaderT[T]) loadInternal() (*Config, error) {
	k := koanf.New(".")

	// 加载默认值（map 形式）
	if l.opts.defaults.IsPresent() {
		defaults, _ := l.opts.defaults.Get()
		if err := k.Load(confmap.Provider(defaults, "."), nil); err != nil {
			return nil, err
		}
	}

	// 加载默认值（struct 形式）
	if l.opts.defaultsStruct != nil {
		defaultMap, err := structToMap(l.opts.defaultsStruct)
		if err != nil {
			return nil, err
		}
		if err := k.Load(confmap.Provider(defaultMap, "."), nil); err != nil {
			return nil, err
		}
	}

	// 按优先级加载
	for _, src := range l.opts.priority {
		switch src {
		case SourceDotenv:
			if err := loadDotenv(k, l.opts.dotenvFiles, l.opts.ignoreDotenvErr); err != nil {
				return nil, err
			}
		case SourceFile:
			if err := loadFiles(k, l.opts.files); err != nil {
				return nil, err
			}
		case SourceEnv:
			if err := loadEnv(k, l.opts.envPrefix); err != nil {
				return nil, err
			}
		}
	}

	return newConfig(k, l.opts), nil
}

// NewT 创建泛型配置加载器
func NewT[T any](opts ...Option) *LoaderT[T] {
	options := NewOptions()
	for _, opt := range opts {
		opt(options)
	}
	return &LoaderT[T]{opts: options}
}

// Load 便捷函数：直接加载配置到结构体（非泛型）
func Load(out any, opts ...Option) error {
	loader := New(opts...)
	return loader.Load(out)
}

// LoadT 便捷函数：直接加载配置到泛型结构体
func LoadT[T any](opts ...Option) mo.Result[T] {
	loader := NewT[T](opts...)
	return loader.Load()
}

// LoadConfig 便捷函数：直接加载配置并返回 Config 对象
func LoadConfig(opts ...Option) (*Config, error) {
	loader := New(opts...)
	return loader.LoadConfig()
}

// LoadConfigT 便捷函数：直接加载配置并返回 Config 对象（泛型版本）
func LoadConfigT[T any](opts ...Option) (*Config, error) {
	loader := NewT[T](opts...)
	return loader.LoadConfig()
}
