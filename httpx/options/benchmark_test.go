package options_test

import (
	"testing"
	"time"

	options "github.com/DaiYuANg/arcgo/httpx/options"
)

func BenchmarkServerOptionsBuild(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		opts := options.DefaultServerOptions()
		options.Compose(
			options.WithBasePath("/api/v1"),
			options.WithValidation(true),
		)(opts)

		compiled := opts.Build()
		if compiled.Len() == 0 {
			b.Fatal("expected non-empty server options")
		}
	}
}

func BenchmarkHTTPClientOptionsBuild(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		opts := options.DefaultHTTPClientOptions()
		options.WithHTTPTimeout(5 * time.Second)(opts)
		client := opts.Build()
		if client.Timeout != 5*time.Second {
			b.Fatalf("unexpected timeout: %s", client.Timeout)
		}
	}
}

func BenchmarkContextOptionsBuild(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		opts := &options.ContextOptions{
			Timeout: 2 * time.Second,
		}
		options.WithContextValueOpt(opts, "trace_id", "bench-trace")
		ctx, cancel := opts.Build()
		if cancel != nil {
			cancel()
		}
		if ctx == nil {
			b.Fatal("context should not be nil")
		}
	}
}
