package configx

import "testing"

type benchmarkServiceConfig struct {
	Name string
	Port int
}

var benchmarkDefaults = map[string]any{
	"service.name": "arcgo",
	"service.port": 8080,
	"feature.x":    true,
}

func benchmarkLoadedConfig(b *testing.B) *Config {
	b.Helper()

	cfg, err := LoadConfig(WithDefaults(benchmarkDefaults))
	if err != nil {
		b.Fatalf("load config: %v", err)
	}
	return cfg
}

func BenchmarkLoadConfigDefaults(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cfg, err := LoadConfig(WithDefaults(benchmarkDefaults))
		if err != nil {
			b.Fatalf("load config: %v", err)
		}
		if cfg.GetString("service.name") == "" {
			b.Fatal("service.name should not be empty")
		}
	}
}

func BenchmarkConfigGetters(b *testing.B) {
	cfg := benchmarkLoadedConfig(b)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = cfg.GetString("service.name")
		_ = cfg.GetInt("service.port")
		_ = cfg.GetBool("feature.x")
	}
}

func BenchmarkGetAsStruct(b *testing.B) {
	cfg := benchmarkLoadedConfig(b)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		value, err := GetAs[benchmarkServiceConfig](cfg, "service")
		if err != nil {
			b.Fatalf("GetAs failed: %v", err)
		}
		if value.Name == "" || value.Port == 0 {
			b.Fatal("unexpected empty struct from GetAs")
		}
	}
}
