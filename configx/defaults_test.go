package configx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type nestedDefaults struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type appDefaults struct {
	Name string         `mapstructure:"name"`
	DB   nestedDefaults `mapstructure:"db"`
	Skip string         `mapstructure:"-"`
}

func TestStructToMap_Struct(t *testing.T) {
	in := appDefaults{
		Name: "demo",
		DB: nestedDefaults{
			Host: "localhost",
			Port: 5432,
		},
		Skip: "ignore",
	}

	got, err := structToMap(in)
	assert.NoError(t, err)
	assert.Equal(t, "demo", got["name"])

	db, ok := got["db"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "localhost", db["host"])
	assert.Equal(t, 5432, db["port"])

	_, exists := got["-"]
	assert.False(t, exists)
}

func TestStructToMap_MapInput(t *testing.T) {
	in := map[string]interface{}{
		"name": "demo",
		"port": 8080,
	}

	got, err := structToMap(in)
	assert.NoError(t, err)
	assert.Equal(t, "demo", got["name"])
	assert.Equal(t, 8080, got["port"])
}

func TestStructToMap_InvalidType(t *testing.T) {
	_, err := structToMap(123)
	assert.Error(t, err)
}

func TestWithIgnoreDotenvError_DefaultTrue(t *testing.T) {
	var cfg SimpleConfig
	err := Load(&cfg,
		WithDotenv("missing.env"),
		WithPriority(SourceDotenv),
	)
	assert.NoError(t, err)
}
