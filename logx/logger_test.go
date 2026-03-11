package logx

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

type failingCloser struct {
	err error
}

func (c *failingCloser) Close() error {
	return c.err
}

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
		{"trace", LevelTrace},
		{"fatal", LevelFatal},
		{"panic", LevelPanic},
		{"disabled", LevelDisabled},
	}

	for _, tt := range tests {
		got, err := ParseLevel(tt.input)
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
	_ = MustParseLevel("invalid-level")
}

func TestNew_ConfigOfAndClose(t *testing.T) {
	t.Parallel()

	logger, err := New(
		WithConsole(true),
		WithLevel(slog.LevelDebug),
		WithCaller(true),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = Close(logger) }()

	cfg, ok := ConfigOf(logger)
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

	_, err := New(WithFileRotation(0, 7, 10))
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestProductionConfig_WritesFile(t *testing.T) {
	t.Parallel()

	logPath := filepath.Join(t.TempDir(), "app.log")
	logger, err := New(ProductionConfig(logPath)...)
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("production file test", "key", "value")
	if err := Close(logger); err != nil {
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
	logger := slog.New(&managedHandler{
		Handler: slog.NewTextHandler(io.Discard, nil),
		state: &lifecycleState{
			closers: []io.Closer{
				&failingCloser{err: errA},
				nil,
				&failingCloser{err: errB},
			},
		},
	})

	err := Close(logger)
	if err == nil {
		t.Fatal("expected non-nil close error")
	}
	if !errors.Is(err, errA) || !errors.Is(err, errB) {
		t.Fatal("expected joined close errors")
	}

	// idempotent: second close should not panic and should return same join result.
	err2 := Close(logger)
	if err2 == nil {
		t.Fatal("expected non-nil close error on second call")
	}
	if !errors.Is(err2, errA) || !errors.Is(err2, errB) {
		t.Fatal("expected joined close errors on second close")
	}
}

func TestWithTraceContext(t *testing.T) {
	t.Parallel()

	logger, err := New(WithConsole(false))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = Close(logger) }()

	if got := WithTraceContext(logger, context.Background()); got != logger {
		t.Fatal("expected original logger when context has no span")
	}

	traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	derived := WithTraceContext(logger, ctx)
	if derived == logger {
		t.Fatal("expected derived logger when span context exists")
	}
}

func TestFieldHelpers(t *testing.T) {
	t.Parallel()

	logger, err := New(WithConsole(false), WithDebugLevel())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = Close(logger) }()

	if got := WithField(logger, "retry", 3); got == nil {
		t.Fatal("expected non-nil logger")
	}
	if got := WithFields(logger, map[string]any{"batch": 7}); got == nil {
		t.Fatal("expected non-nil logger")
	}
	if got := WithFieldT(logger, "tenant", "acme"); got == nil {
		t.Fatal("expected non-nil logger")
	}
	if got := WithFieldsT(logger, map[string]int{"attempt": 2}); got == nil {
		t.Fatal("expected non-nil logger")
	}
	if got := WithError(logger, errors.New("boom")); got == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestOopsHelpers(t *testing.T) {
	t.Parallel()

	logger, err := New(WithConsole(false))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = Close(logger) }()

	LogOops(logger, errors.New("boom"))
	LogOopsWithStack(logger, errors.New("boom"))

	if Oops() == nil {
		t.Fatal("expected non-nil oops error")
	}
	if got := Oopsf("user.%s", "not_found"); got == nil || !strings.Contains(got.Error(), "user.not_found") {
		t.Fatal("expected formatted oops error")
	}
	if OopsWith(context.Background()) == nil {
		t.Fatal("expected non-nil context oops error")
	}
}

func TestLevelOfAndEnabled(t *testing.T) {
	t.Parallel()

	logger, err := New(WithLevel(slog.LevelWarn), WithConsole(false))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = Close(logger) }()

	level, ok := LevelOf(logger)
	if !ok {
		t.Fatal("expected level metadata")
	}
	if level != slog.LevelWarn {
		t.Fatalf("expected warn level, got %v", level)
	}

	if IsEnabled(logger, slog.LevelDebug) {
		t.Fatal("debug should be disabled at warn level")
	}
	if !IsEnabled(logger, slog.LevelError) {
		t.Fatal("error should be enabled at warn level")
	}
}
