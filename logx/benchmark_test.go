package logx

import (
	"log/slog"
	"testing"
)

func benchmarkLogger(b *testing.B) *slog.Logger {
	b.Helper()

	logger, err := New(WithConsole(false), WithDebugLevel())
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		_ = Close(logger)
	})
	return logger
}

func BenchmarkLoggerInfo(b *testing.B) {
	logger := benchmarkLogger(b)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "key", "value", "count", i)
	}
}

func BenchmarkLoggerWithFieldsInfo(b *testing.B) {
	logger := benchmarkLogger(b)
	fields := map[string]interface{}{
		"service": "arcgo",
		"env":     "bench",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		WithFields(logger, fields).Info("with-fields")
	}
}

func BenchmarkSlogInfo(b *testing.B) {
	slogLogger := benchmarkLogger(b)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		slogLogger.Info("slog benchmark", "key", "value", "count", i)
	}
}
