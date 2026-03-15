package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/DaiYuANg/archgo/configx"
)

type AppConfig struct {
	Version string `mapstructure:"version" validate:"required"`
	HTTP    struct {
		Addr string `mapstructure:"addr" validate:"required"`
	} `mapstructure:"http" validate:"required"`
	Base struct {
		Path string `mapstructure:"path" validate:"required"`
	} `mapstructure:"base" validate:"required"`
	Docs struct {
		Path string `mapstructure:"path" validate:"required"`
	} `mapstructure:"docs" validate:"required"`
	OpenAPI struct {
		Path string `mapstructure:"path" validate:"required"`
	} `mapstructure:"openapi" validate:"required"`
	Metrics struct {
		Path string `mapstructure:"path" validate:"required"`
	} `mapstructure:"metrics" validate:"required"`
	DB struct {
		Driver string `mapstructure:"driver" validate:"required,oneof=sqlite mysql postgres"`
		DSN    string `mapstructure:"dsn" validate:"required"`
	} `mapstructure:"db" validate:"required"`
	JWT struct {
		Secret  string `mapstructure:"secret" validate:"required,min=8"`
		Issuer  string `mapstructure:"issuer" validate:"required"`
		Expires struct {
			Minutes int `mapstructure:"minutes" validate:"required,min=1,max=10080"`
		} `mapstructure:"expires" validate:"required"`
	} `mapstructure:"jwt" validate:"required"`
	Event struct {
		Workers  int  `mapstructure:"workers" validate:"required,min=1,max=4096"`
		Parallel bool `mapstructure:"parallel"`
	} `mapstructure:"event" validate:"required"`
}

func New() (AppConfig, error) {
	cfg, err := configx.LoadTErr[AppConfig](
		configx.WithEnvPrefix("RBAC"),
		configx.WithDefaultsFrom(defaultAppConfig()),
		configx.WithValidateLevel(configx.ValidateLevelRequired),
	)
	if err != nil {
		return AppConfig{}, fmt.Errorf("load app config: %w", err)
	}
	return cfg, nil
}

func defaultAppConfig() AppConfig {
	cfg := AppConfig{}
	cfg.Version = "0.4.0"
	cfg.HTTP.Addr = ":18080"
	cfg.Base.Path = "/api/v1"
	cfg.Docs.Path = "/docs"
	cfg.OpenAPI.Path = "/openapi.json"
	cfg.Metrics.Path = "/metrics"
	cfg.DB.Driver = "sqlite"
	cfg.DB.DSN = "file:rbac_basic.db?cache=shared"
	cfg.JWT.Secret = "change-me-in-production"
	cfg.JWT.Issuer = "arcgo-rbac-example"
	cfg.JWT.Expires.Minutes = 120
	cfg.Event.Workers = 8
	cfg.Event.Parallel = true
	return cfg
}

func (c AppConfig) Addr() string {
	return c.HTTP.Addr
}

func (c AppConfig) BasePath() string {
	return c.Base.Path
}

func (c AppConfig) DocsPath() string {
	return c.Docs.Path
}

func (c AppConfig) OpenAPIPath() string {
	return c.OpenAPI.Path
}

func (c AppConfig) MetricsPath() string {
	return c.Metrics.Path
}

func (c AppConfig) DBDSN() string {
	return c.DB.DSN
}

func (c AppConfig) DBDriver() string {
	return strings.ToLower(strings.TrimSpace(c.DB.Driver))
}

func (c AppConfig) JWTExpiresIn() time.Duration {
	return time.Duration(c.JWT.Expires.Minutes) * time.Minute
}
