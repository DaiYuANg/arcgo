package logx

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/rs/zerolog"
)

const (
	// LevelTrace is the trace level for slog-compatible logger configuration.
	LevelTrace slog.Level = slog.LevelDebug - 4
	// LevelFatal is the fatal level for slog-compatible logger configuration.
	LevelFatal slog.Level = slog.LevelError + 4
	// LevelPanic is the panic level for slog-compatible logger configuration.
	LevelPanic slog.Level = slog.LevelError + 8
	// LevelDisabled effectively disables log emission for slog-compatible logger configuration.
	LevelDisabled slog.Level = slog.Level(127)
)

const (
	// TraceLevel is a backward-compatible alias of LevelTrace.
	TraceLevel = LevelTrace
	// DebugLevel is a backward-compatible alias of slog.LevelDebug.
	DebugLevel = slog.LevelDebug
	// InfoLevel is a backward-compatible alias of slog.LevelInfo.
	InfoLevel = slog.LevelInfo
	// WarnLevel is a backward-compatible alias of slog.LevelWarn.
	WarnLevel = slog.LevelWarn
	// ErrorLevel is a backward-compatible alias of slog.LevelError.
	ErrorLevel = slog.LevelError
	// FatalLevel is a backward-compatible alias of LevelFatal.
	FatalLevel = LevelFatal
	// PanicLevel is a backward-compatible alias of LevelPanic.
	PanicLevel = LevelPanic
	// DisabledLevel is a backward-compatible alias of LevelDisabled.
	DisabledLevel = LevelDisabled
)

// ParseLevel parses a level string into slog.Level.
func ParseLevel(input string) (slog.Level, error) {
	text := strings.TrimSpace(strings.ToLower(input))
	switch text {
	case "trace":
		return LevelTrace, nil
	case "fatal":
		return LevelFatal, nil
	case "panic":
		return LevelPanic, nil
	case "disabled", "disable", "off":
		return LevelDisabled, nil
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(text)); err != nil {
		return slog.LevelInfo, fmt.Errorf("invalid log level %q: %w", input, err)
	}
	return level, nil
}

// MustParseLevel parses a level string and panics on error.
func MustParseLevel(input string) slog.Level {
	level, err := ParseLevel(input)
	if err != nil {
		panic(err)
	}
	return level
}

func toZerologLevel(level slog.Level) zerolog.Level {
	switch {
	case level <= LevelTrace:
		return zerolog.TraceLevel
	case level <= slog.LevelDebug:
		return zerolog.DebugLevel
	case level <= slog.LevelInfo:
		return zerolog.InfoLevel
	case level <= slog.LevelWarn:
		return zerolog.WarnLevel
	case level <= slog.LevelError:
		return zerolog.ErrorLevel
	case level <= LevelFatal:
		return zerolog.FatalLevel
	case level <= LevelPanic:
		return zerolog.PanicLevel
	case level >= LevelDisabled:
		return zerolog.Disabled
	default:
		return zerolog.InfoLevel
	}
}
