package logx_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DaiYuANg/arcgo/logx"
	"go.opentelemetry.io/otel/trace"
)

func TestParseLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"trace", logx.LevelTrace},
		{"fatal", logx.LevelFatal},
		{"panic", logx.LevelPanic},
		{"disabled", logx.LevelDisabled},
	}

	for _, tt := range tests {
		got, err := logx.ParseLevel(tt.input)
		if err != nil {
			t.Fatalf("ParseLevel(%q) error = %v", tt.input, err)
		}
		if got != tt.want {
			t.Fatalf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestMustParseLevel_PanicOnInvalid(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = logx.MustParseLevel("invalid-level")
}

func TestNew_ConfigOfAndClose(t *testing.T) {
	t.Parallel()

	logger := newTestLogger(t,
		logx.WithConsole(true),
		logx.WithLevel(slog.LevelDebug),
		logx.WithCaller(true),
	)

	cfg, ok := logx.ConfigOf(logger)
	if !ok {
		t.Fatal("expected config to be available")
	}
	if cfg.Level != slog.LevelDebug {
		t.Fatalf("expected level debug, got %v", cfg.Level)
	}
	if !cfg.AddCaller {
		t.Fatal("expected AddCaller=true")
	}
}

func TestNew_InvalidRotationConfig(t *testing.T) {
	t.Parallel()

	_, err := logx.New(logx.WithFileRotation(0, 7, 10))
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestProductionConfig_WritesFile(t *testing.T) {
	t.Parallel()

	logPath := filepath.Join(t.TempDir(), "app.log")
	logger, err := logx.New(logx.ProductionConfig(logPath)...)
	if err != nil {
		t.Fatal(err)
	}

	logger.Info("production file test", "key", "value")
	if err := logx.Close(logger); err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(logPath); statErr != nil {
		t.Fatalf("expected log file to exist: %v", statErr)
	}
}

func TestClose_JoinedErrorAndIdempotent(t *testing.T) {
	t.Parallel()

	errA := errors.New("close-a")
	errB := errors.New("close-b")
	logger := slog.New(&closableHandler{
		Handler: slog.DiscardHandler,
		closers: []simpleCloser{
			&failingCloser{err: errA},
			nil,
			&failingCloser{err: errB},
		},
	})

	err := logx.Close(logger)
	if err == nil {
		t.Fatal("expected non-nil close error")
	}
	if !errors.Is(err, errA) || !errors.Is(err, errB) {
		t.Fatal("expected joined close errors")
	}

	err2 := logx.Close(logger)
	if err2 == nil {
		t.Fatal("expected non-nil close error on second call")
	}
	if !errors.Is(err2, errA) || !errors.Is(err2, errB) {
		t.Fatal("expected joined close errors on second close")
	}
}

func TestWithTraceContext(t *testing.T) {
	t.Parallel()

	logger := newTestLogger(t, logx.WithConsole(false))

	if got := logx.WithTraceContext(context.Background(), logger); got != logger {
		t.Fatal("expected original logger when context has no span")
	}

	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	if err != nil {
		t.Fatal(err)
	}
	spanID, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	if err != nil {
		t.Fatal(err)
	}
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	derived := logx.WithTraceContext(ctx, logger)
	if derived == logger {
		t.Fatal("expected derived logger when span context exists")
	}
}

func TestFieldHelpers(t *testing.T) {
	t.Parallel()

	logger := newTestLogger(t, logx.WithConsole(false), logx.WithDebugLevel())

	if got := logx.WithField(logger, "retry", 3); got == nil {
		t.Fatal("expected non-nil logger")
	}
	if got := logx.WithFields(logger, map[string]any{"batch": 7}); got == nil {
		t.Fatal("expected non-nil logger")
	}
	if got := logx.WithFieldT(logger, "tenant", "acme"); got == nil {
		t.Fatal("expected non-nil logger")
	}
	if got := logx.WithFieldsT(logger, map[string]int{"attempt": 2}); got == nil {
		t.Fatal("expected non-nil logger")
	}
	if got := logx.WithError(logger, errors.New("boom")); got == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestOopsHelpers(t *testing.T) {
	t.Parallel()

	logger := newTestLogger(t, logx.WithConsole(false))

	logx.LogOops(logger, errors.New("boom"))
	logx.LogOopsWithStack(logger, errors.New("boom"))

	if logx.Oops() == nil {
		t.Fatal("expected non-nil oops error")
	}
	if got := logx.Oopsf("user.%s", "not_found"); got == nil || !strings.Contains(got.Error(), "user.not_found") {
		t.Fatal("expected formatted oops error")
	}
	if logx.OopsWith(context.Background()) == nil {
		t.Fatal("expected non-nil context oops error")
	}
}

func TestLevelOfAndEnabled(t *testing.T) {
	t.Parallel()

	logger := newTestLogger(t, logx.WithLevel(slog.LevelWarn), logx.WithConsole(false))

	level, ok := logx.LevelOf(logger)
	if !ok {
		t.Fatal("expected level metadata")
	}
	if level != slog.LevelWarn {
		t.Fatalf("expected warn level, got %v", level)
	}

	if logx.IsEnabled(logger, slog.LevelDebug) {
		t.Fatal("debug should be disabled at warn level")
	}
	if !logx.IsEnabled(logger, slog.LevelError) {
		t.Fatal("error should be enabled at warn level")
	}
}
