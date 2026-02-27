package configx

// NewConfig 创建并加载配置实例。
// 使用 Option 模式传入配置源（文件、环境变量、默认值等）。
// 返回 *Config 实例和错误。配置加载失败时返回 nil 和 error.
//
// 示例：
//
//	cfg, err := configx.NewConfig(
//	    configx.WithFiles("config.yaml"),
//	    configx.WithEnvPrefix("APP_"),
//	    configx.WithDefaults(map[string]any{"port": 8080}),
//	)
func NewConfig(opts ...Option) (*Config, error) {
	return LoadConfig(opts...)
}
