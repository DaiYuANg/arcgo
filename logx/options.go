package logx

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
)

// Option documents related behavior.
type Option func(*BuildOptions)

// BuildOptions configures logger construction.
type BuildOptions struct {
	level      slog.Level
	console    bool
	noColor    bool
	filePath   string
	maxSize    int
	maxAge     int
	maxBackups int
	timeFormat string
	setGlobal  bool
	addCaller  bool
	localTime  bool
	compress   bool
}

type config = BuildOptions

// Config is an exported read-only snapshot of logger build options.
type Config struct {
	Level      slog.Level
	Console    bool
	NoColor    bool
	FilePath   string
	MaxSizeMB  int
	MaxAgeDays int
	MaxBackups int
	TimeFormat string
	SetGlobal  bool
	AddCaller  bool
	LocalTime  bool
	Compress   bool
}

// defaultConfig provides default behavior.
func defaultConfig() BuildOptions {
	return BuildOptions{
		level:      slog.LevelInfo,
		console:    true,
		noColor:    false,
		timeFormat: "2006-01-02 15:04:05",
		maxSize:    100,  // 100MB
		maxAge:     7,    // 7 days
		maxBackups: 10,   // 10 files
		localTime:  true, // use local time for rotation
		compress:   true, // compress rotated files
	}
}

// validate documents related behavior.
func (c *BuildOptions) validate() error {
	if c.maxSize < 1 {
		return fmt.Errorf("maxSize must be at least 1MB, got %d", c.maxSize)
	}

	if c.maxAge < 0 {
		return fmt.Errorf("maxAge cannot be negative, got %d", c.maxAge)
	}

	if c.maxBackups < 0 {
		return fmt.Errorf("maxBackups cannot be negative, got %d", c.maxBackups)
	}

	return nil
}

func (c BuildOptions) export() Config {
	return Config{
		Level:      c.level,
		Console:    c.console,
		NoColor:    c.noColor,
		FilePath:   c.filePath,
		MaxSizeMB:  c.maxSize,
		MaxAgeDays: c.maxAge,
		MaxBackups: c.maxBackups,
		TimeFormat: c.timeFormat,
		SetGlobal:  c.setGlobal,
		AddCaller:  c.addCaller,
		LocalTime:  c.localTime,
		Compress:   c.compress,
	}
}

// Note.

// WithLevel configures logger level using slog.Level.
func WithLevel(level slog.Level) Option {
	return func(c *config) {
		c.level = level
	}
}

// WithLevelString configures related behavior.
func WithLevelString(level string) Option {
	return func(c *config) {
		l, err := ParseLevel(level)
		if err == nil {
			c.level = l
		}
	}
}

// WithTraceLevel enables related functionality.
func WithTraceLevel() Option {
	return WithLevel(LevelTrace)
}

// WithDebugLevel enables related functionality.
func WithDebugLevel() Option {
	return WithLevel(slog.LevelDebug)
}

// WithInfoLevel enables related functionality.
func WithInfoLevel() Option {
	return WithLevel(slog.LevelInfo)
}

// WithWarnLevel enables related functionality.
func WithWarnLevel() Option {
	return WithLevel(slog.LevelWarn)
}

// WithErrorLevel enables related functionality.
func WithErrorLevel() Option {
	return WithLevel(slog.LevelError)
}

// WithFatalLevel enables related functionality.
func WithFatalLevel() Option {
	return WithLevel(LevelFatal)
}

// WithPanicLevel enables related functionality.
func WithPanicLevel() Option {
	return WithLevel(LevelPanic)
}

// Note.

// WithConsole enables related functionality.
func WithConsole(enabled bool) Option {
	return func(c *config) {
		c.console = enabled
	}
}

// WithNoColor disables related functionality.
func WithNoColor() Option {
	return func(c *config) {
		c.noColor = true
	}
}

// WithFile configures related behavior.
// Note.
func WithFile(path string) Option {
	return func(c *config) {
		c.filePath = path
	}
}

// WithFileRotation documents related behavior.
// maxSize documents related behavior.
// maxAge documents related behavior.
// maxBackups documents related behavior.
func WithFileRotation(maxSizeMB, maxAgeDays, maxBackups int) Option {
	return func(c *config) {
		c.maxSize = maxSizeMB
		c.maxAge = maxAgeDays
		c.maxBackups = maxBackups
	}
}

// WithLocalTime documents related behavior.
func WithLocalTime(enabled bool) Option {
	return func(c *config) {
		c.localTime = enabled
	}
}

// WithCompress enables related functionality.
func WithCompress(enabled bool) Option {
	return func(c *config) {
		c.compress = enabled
	}
}

// Note.

// WithTimeFormat configures related behavior.
// Note.
func WithTimeFormat(format string) Option {
	return func(c *config) {
		c.timeFormat = format
	}
}

// WithRFC3339Time documents related behavior.
func WithRFC3339Time() Option {
	return WithTimeFormat(time.RFC3339)
}

// WithISO8601Time documents related behavior.
func WithISO8601Time() Option {
	return WithTimeFormat("2006-01-02T15:04:05Z07:00")
}

// Note.

// WithGlobalLogger configures related behavior.
func WithGlobalLogger() Option {
	return func(c *config) {
		c.setGlobal = true
	}
}

// WithCaller enables related functionality.
func WithCaller(enabled bool) Option {
	return func(c *config) {
		c.addCaller = enabled
	}
}

// Note.

// DevelopmentConfig documents related behavior.
// Note.
// Note.
// Note.
func DevelopmentConfig() collectionx.List[Option] {
	return collectionx.NewList(
		WithConsole(true),
		WithDebugLevel(),
		WithCaller(true),
	)
}

// ProductionConfig documents related behavior.
// Note.
// Note.
// Note.
func ProductionConfig(logPath string) collectionx.List[Option] {
	return collectionx.NewList(
		WithConsole(false),
		WithInfoLevel(),
		WithFile(logPath),
		WithFileRotation(100, 7, 10),
		WithCompress(true),
	)
}
