package configx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

type SimpleConfig struct {
	Name string `validate:"required"`
	Port int    `validate:"gte=1000,lte=65535"`
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

func TestWithDefaultsTyped(t *testing.T) {
	cfg, err := LoadConfig(
		WithDefaultsTyped(map[string]int{
			"port": 7001,
		}),
	)
	assert.NoError(t, err)
	assert.Equal(t, 7001, cfg.GetInt("port"))
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

func TestLoadTErr_Generic(t *testing.T) {
	cfg, err := LoadTErr[SimpleConfig](
		WithDefaults(map[string]any{
			"name": "tuple",
			"port": 9100,
		}),
	)
	assert.NoError(t, err)
	assert.Equal(t, "tuple", cfg.Name)
	assert.Equal(t, 9100, cfg.Port)
}

func TestWithTypedDefaults_Generic(t *testing.T) {
	type AppConfig struct {
		Name string `validate:"required"`
		Port int    `validate:"gte=1"`
	}

	cfg, err := LoadTErr[AppConfig](
		WithTypedDefaults(AppConfig{Name: "typed-default", Port: 8081}),
		WithValidateLevel(ValidateLevelStruct),
	)
	assert.NoError(t, err)
	assert.Equal(t, "typed-default", cfg.Name)
	assert.Equal(t, 8081, cfg.Port)
}

func TestSnapshot_ReturnsSortedKeys(t *testing.T) {
	cfg, err := LoadConfig(
		WithDefaults(map[string]any{
			"b.key": 2,
			"a.key": 1,
		}),
		WithPriority(),
	)
	assert.NoError(t, err)
	snapshot := cfg.Snapshot()
	assert.Equal(t, []string{"a.key", "b.key"}, snapshot.Keys)
	assert.Equal(t, 1, snapshot.Values["a.key"])
	assert.Equal(t, 2, snapshot.Values["b.key"])
}

func TestValidate_Required(t *testing.T) {
	result := LoadT[SimpleConfig](
		WithDefaults(map[string]any{
			"name": "", // empty → required fails
			"port": 8080,
		}),
		WithValidateLevel(ValidateLevelStruct),
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
		WithValidateLevel(ValidateLevelStruct),
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
			"app.ids":     []int{1, 2, 3},
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
	assert.Equal(t, int64(1234), cfg.GetInt64("app.port"))
	assert.Equal(t, []int{1, 2, 3}, cfg.GetIntSlice("app.ids"))
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

func TestDotenvDefaultModeIsOptional(t *testing.T) {
	var cfg SimpleConfig
	err := Load(&cfg,
		WithDotenv("not-exists.env"),
		WithPriority(SourceDotenv),
	)
	assert.NoError(t, err)
}

func TestWithIgnoreDotenvError_IgnoreParseError(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), ".env")
	writeErr := os.WriteFile(envFile, []byte("BROKEN='unclosed"), 0o600)
	assert.NoError(t, writeErr)

	var cfg SimpleConfig
	err := Load(&cfg,
		WithDotenv(envFile),
		WithIgnoreDotenvError(true),
		WithPriority(SourceDotenv),
	)
	assert.NoError(t, err)
}

func TestWithIgnoreDotenvError_StrictParseError(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), ".env")
	writeErr := os.WriteFile(envFile, []byte("BROKEN='unclosed"), 0o600)
	assert.NoError(t, writeErr)

	var cfg SimpleConfig
	err := Load(&cfg,
		WithDotenv(envFile),
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

func TestGetAs_GenericValue(t *testing.T) {
	cfg, err := LoadConfig(
		WithDefaults(map[string]any{
			"service.port": 9090,
			"service.name": "arcgo",
		}),
	)
	assert.NoError(t, err)

	port, err := GetAs[int](cfg, "service.port")
	assert.NoError(t, err)
	assert.Equal(t, 9090, port)

	name, err := GetAs[string](cfg, "service.name")
	assert.NoError(t, err)
	assert.Equal(t, "arcgo", name)
}

func TestGetAsOr_And_MustGetAs(t *testing.T) {
	cfg, err := LoadConfig(
		WithDefaults(map[string]any{
			"service.port": 9090,
		}),
	)
	assert.NoError(t, err)

	got := GetAsOr[int](cfg, "service.missing", 8080)
	assert.Equal(t, 8080, got)

	assert.Equal(t, 9090, MustGetAs[int](cfg, "service.port"))
}
