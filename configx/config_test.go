package configx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type SimpleConfig struct {
	Name string `mapstructure:"name" validate:"required"`
	Port int    `mapstructure:"port" validate:"gte=1000,lte=65535"`
}

func TestNewConfig_Basic(t *testing.T) {
	cfg, err := NewConfig(
		WithDefaults(map[string]any{
			"name": "test",
			"port": 8080,
		}),
	)
	assert.NoError(t, err)
	assert.Equal(t, "test", cfg.GetString("name"))
	assert.Equal(t, 8080, cfg.GetInt("port"))
}

func TestWithDefaultsStruct(t *testing.T) {
	defaults := SimpleConfig{Name: "struct-default", Port: 3000}
	cfg, err := LoadConfig(
		WithDefaultsStruct(defaults),
	)
	assert.NoError(t, err)
	assert.Equal(t, "struct-default", cfg.GetString("name"))
	assert.Equal(t, 3000, cfg.GetInt("port"))
}

func TestLoadT_Generic(t *testing.T) {
	result := LoadT[SimpleConfig](
		WithDefaults(map[string]any{
			"name": "gen",
			"port": 9000,
		}),
	)
	assert.True(t, result.IsOk())
	cfg, err := result.Get()
	assert.NoError(t, err)
	assert.Equal(t, "gen", cfg.Name)
	assert.Equal(t, 9000, cfg.Port)
}

func TestValidate_Required(t *testing.T) {
	result := LoadT[SimpleConfig](
		WithDefaults(map[string]any{
			"name": "", // empty → required fails
			"port": 8080,
		}),
		WithValidateLevel(ValidateLevelRequired),
	)
	assert.True(t, result.IsError())
	err := result.Error()
	assert.Error(t, err)
}

func TestValidate_Range(t *testing.T) {
	result := LoadT[SimpleConfig](
		WithDefaults(map[string]any{
			"name": "ok",
			"port": 500, // < 1000 → gte fails
		}),
		WithValidateLevel(ValidateLevelRequired),
	)
	assert.True(t, result.IsError())
	assert.Error(t, result.Error())
}

func TestGetters(t *testing.T) {
	cfg, err := LoadConfig(
		WithDefaults(map[string]any{
			"app.name":    "getter-test",
			"app.port":    1234,
			"app.debug":   true,
			"app.timeout": "5s",
			"app.tags":    []string{"x", "y"},
			"app.ratio":   0.75,
		}),
	)
	assert.NoError(t, err)

	assert.Equal(t, "getter-test", cfg.GetString("app.name"))
	assert.Equal(t, 1234, cfg.GetInt("app.port"))
	assert.True(t, cfg.GetBool("app.debug"))
	assert.Equal(t, 5, int(cfg.GetDuration("app.timeout").Seconds()))
	assert.Equal(t, []string{"x", "y"}, cfg.GetStringSlice("app.tags"))
	assert.Equal(t, 0.75, cfg.GetFloat64("app.ratio"))
	assert.True(t, cfg.Exists("app.name"))
	assert.False(t, cfg.Exists("missing"))
}

func TestWithIgnoreDotenvError(t *testing.T) {
	var cfg SimpleConfig
	err := Load(&cfg,
		WithDotenv("not-exists.env"),
		WithIgnoreDotenvError(false),
		WithPriority(SourceDotenv),
	)
	assert.Error(t, err)
}

func TestEnvPrefixWithoutTrailingUnderscore(t *testing.T) {
	t.Setenv("APP_NAME", "env-app")
	t.Setenv("APP_PORT", "8088")

	result := LoadT[SimpleConfig](
		WithEnvPrefix("APP"),
		WithPriority(SourceEnv),
	)
	assert.True(t, result.IsOk())

	cfg, err := result.Get()
	assert.NoError(t, err)
	assert.Equal(t, "env-app", cfg.Name)
	assert.Equal(t, 8088, cfg.Port)
}
