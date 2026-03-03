package logx

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/samber/oops"
)

func TestLevel(t *testing.T) {
	// 测试 String 方法
	tests := []struct {
		level    Level
		expected string
	}{
		{TraceLevel, "trace"},
		{DebugLevel, "debug"},
		{InfoLevel, "info"},
		{WarnLevel, "warn"},
		{ErrorLevel, "error"},
		{FatalLevel, "fatal"},
		{PanicLevel, "panic"},
		{DisabledLevel, "disabled"},
		{NoLevel, "none"},
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			if tt.level.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.level.String())
			}
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
		hasError bool
	}{
		{"trace", TraceLevel, false},
		{"debug", DebugLevel, false},
		{"info", InfoLevel, false},
		{"warn", WarnLevel, false},
		{"warning", WarnLevel, false},
		{"error", ErrorLevel, false},
		{"fatal", FatalLevel, false},
		{"panic", PanicLevel, false},
		{"disabled", DisabledLevel, false},
		{"none", NoLevel, false},
		{"TRACE", TraceLevel, false},
		{"DEBUG", DebugLevel, false},
		{"invalid", NoLevel, true},
		{"", NoLevel, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level, err := ParseLevel(tt.input)
			if (err != nil) != tt.hasError {
				t.Errorf("ParseLevel(%q) error = %v, hasError = %v", tt.input, err, tt.hasError)
			}
			if level != tt.expected {
				t.Errorf("ParseLevel(%q) = %v, expected %v", tt.input, level, tt.expected)
			}
		})
	}
}

func TestMustParseLevel(t *testing.T) {
	// 测试正常情况
	level := MustParseLevel("debug")
	if level != DebugLevel {
		t.Errorf("expected DebugLevel, got %v", level)
	}

	// 测试 panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParseLevel should panic with invalid level")
		}
	}()
	_ = MustParseLevel("invalid")
}

func TestLevelConversion(t *testing.T) {
	levels := []Level{
		TraceLevel, DebugLevel, InfoLevel, WarnLevel,
		ErrorLevel, FatalLevel, PanicLevel, DisabledLevel,
	}

	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			zlLevel := level.ToZerologLevel()
			// 验证转换后的级别可以正确解析
			str := level.String()
			parsed, _ := ParseLevel(str)
			if parsed != level {
				t.Errorf("round-trip failed for %v", level)
			}
			_ = zlLevel
		})
	}
}

func TestLevelEnabled(t *testing.T) {
	tests := []struct {
		level   Level
		current Level
		want    bool
	}{
		{ErrorLevel, InfoLevel, true},
		{InfoLevel, InfoLevel, true},
		{DebugLevel, InfoLevel, false},
		{TraceLevel, ErrorLevel, false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.level, tt.current), func(t *testing.T) {
			if got := tt.level.Enabled(tt.current); got != tt.want {
				t.Errorf("Level.Enabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLogger(t *testing.T) {
	logger, err := New(WithConsole(true))
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	// 测试基本日志
	logger.Info("test info message")
	logger.Debug("test debug message")
	logger.Warn("test warn message")
	logger.Error("test error message")
}

func TestLoggerWithFields(t *testing.T) {
	logger, err := New(WithConsole(true), WithLevel(DebugLevel))
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	// 测试带字段日志
	logger.WithField("user_id", "123").Info("user action")
	logger.WithFields(map[string]any{
		"user_id":   "456",
		"action":    "login",
		"timestamp": time.Now(),
	}).Info("user login")
}

func TestLoggerWithError(t *testing.T) {
	logger, err := New(WithConsole(true))
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	// 测试带 error 日志
	err = os.ErrNotExist
	logger.WithError(err).Error("file not found")
}

func TestLoggerWithContext(t *testing.T) {
	logger, err := New(WithConsole(true), WithLevel(DebugLevel))
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	ctx := context.Background()
	l := logger.WithContext(ctx).Logger()
	l.Info().Msg("context log")
}

func TestSlogIntegration(t *testing.T) {
	logger, err := New(WithConsole(true), WithLevel(DebugLevel))
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	// 测试 slog 集成
	slogLogger := NewSlog(logger)
	slogLogger.Info("slog info message")
	slogLogger.Debug("slog debug message")
	slogLogger.Error("slog error message", "error", "test error")
}

func TestDevelopmentConfig(t *testing.T) {
	logger, err := New(DevelopmentConfig()...)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	logger.Debug("development mode debug")
	logger.Info("development mode info")
}

func TestProductionConfig(t *testing.T) {
	// 使用临时文件测试
	tmpFile := t.TempDir() + "/test.log"
	logger, err := New(ProductionConfig(tmpFile)...)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	logger.Info("production mode info")
	logger.Error("production mode error")

	// 验证文件是否存在
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("log file should exist")
	}
}

func TestMustNew(t *testing.T) {
	// 测试正常情况
	logger := MustNew(WithConsole(true))
	defer logger.Close()
	logger.Info("must new test")

	// 测试错误配置（应该 panic）
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustNew should panic with invalid config")
		}
	}()
	_ = MustNew(WithFileRotation(0, 7, 10)) // invalid maxSize
}

func TestLoggerLevels(t *testing.T) {
	tests := []struct {
		name  string
		level Level
	}{
		{"trace", TraceLevel},
		{"debug", DebugLevel},
		{"info", InfoLevel},
		{"warn", WarnLevel},
		{"error", ErrorLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(WithConsole(true), WithLevel(tt.level))
			if err != nil {
				t.Fatal(err)
			}
			defer logger.Close()

			if logger.GetLevel() != tt.level {
				t.Errorf("expected level %s, got %s", tt.level.String(), logger.GetLevel().String())
			}
		})
	}
}

func TestLoggerConvenienceMethods(t *testing.T) {
	logger, err := New(WithConsole(true), WithLevel(DebugLevel))
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	// 测试便捷方法
	logger.Debug("debug message", "key", "value")
	logger.Info("info message", "key", "value")
	logger.Warn("warn message", "key", "value")
	logger.Error("error message", "key", "value")
}

func TestLoggerConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "invalid maxSize",
			opts:    []Option{WithFileRotation(0, 7, 10)},
			wantErr: true,
		},
		{
			name:    "invalid maxAge",
			opts:    []Option{WithFileRotation(100, -1, 10)},
			wantErr: true,
		},
		{
			name:    "valid config with info level",
			opts:    []Option{WithLevel(InfoLevel), WithConsole(true)},
			wantErr: false,
		},
		{
			name:    "valid config with debug level",
			opts:    []Option{WithLevel(DebugLevel), WithConsole(true)},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOopsIntegration(t *testing.T) {
	logger, err := New(WithConsole(true), WithLevel(DebugLevel))
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	// 测试创建 oops 错误
	err = oops.New("user.not_found")

	// 测试记录 oops 错误
	logger.LogOops(err)

	// 测试 Logger.Oops
	err = logger.Oops()
	if err == nil {
		t.Error("Expected error")
	}

	// 测试 Logger.OopsWith
	ctx := context.Background()
	err = logger.OopsWith(ctx)
	if err == nil {
		t.Error("Expected error")
	}
}

func TestOopsf(t *testing.T) {
	logger, err := New(WithConsole(true), WithLevel(DebugLevel))
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	oopsErr := logger.Oopsf("user.%s", "not_found")
	if oopsErr == nil {
		t.Fatal("expected non-nil error")
	}
	if !strings.Contains(oopsErr.Error(), "user.not_found") {
		t.Fatalf("unexpected error message: %s", oopsErr.Error())
	}
}

func TestWithCallerOption(t *testing.T) {
	logger1, err := New(WithConsole(true), WithCaller(false))
	if err != nil {
		t.Fatal(err)
	}
	defer logger1.Close()
	if logger1.Config().addCaller {
		t.Fatal("expected caller to be disabled")
	}

	logger2, err := New(WithConsole(true), WithCaller(true))
	if err != nil {
		t.Fatal(err)
	}
	defer logger2.Close()
	if !logger2.Config().addCaller {
		t.Fatal("expected caller to be enabled")
	}
}
