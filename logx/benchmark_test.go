package logx_test

import (
	"log/slog"
	"testing"

	"github.com/DaiYuANg/arcgo/logx"
)

func benchmarkLogger(b *testing.B) *slog.Logger {
	b.Helper()

	return newTestLogger(b, logx.WithConsole(false), logx.WithDebugLevel())
}

func BenchmarkLoggerInfo(b *testing.B) {
	logger := benchmarkLogger(b)

	b.ReportAllocs()
	b.ResetTimer()

	for i := range b.N {
		logger.Info("benchmark message", "key", "value", "count", i)
	}
}

func BenchmarkLoggerWithFieldsInfo(b *testing.B) {
	logger := benchmarkLogger(b)
	fields := map[string]any{
		"service": "arcgo",
		"env":     "bench",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		logx.WithFields(logger, fields).Info("with-fields")
	}
}

func BenchmarkSlogInfo(b *testing.B) {
	slogLogger := benchmarkLogger(b)

	b.ReportAllocs()
	b.ResetTimer()

	for i := range b.N {
		slogLogger.Info("slog benchmark", "key", "value", "count", i)
	}
}
