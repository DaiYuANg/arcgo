package logx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/pkg/option"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"github.com/samber/lo"
	oopszerolog "github.com/samber/oops/loggers/zerolog"
	slogzerolog "github.com/samber/slog-zerolog/v2"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/natefinch/lumberjack.v2"
)

var oopsMarshalerOnce sync.Once

type lifecycleState struct {
	cfg     config
	closers []io.Closer

	closeOnce sync.Once
	closeErr  error
}

func (s *lifecycleState) close() error {
	if s == nil {
		return nil
	}

	s.closeOnce.Do(func() {
		errs := lo.FilterMap(s.closers, func(closer io.Closer, _ int) (error, bool) {
			if closer == nil {
				return nil, false
			}
			err := closer.Close()
			if err == nil {
				return nil, false
			}
			return fmt.Errorf("close logger resource: %w", err), true
		})
		s.closeErr = errors.Join(errs...)
	})

	return s.closeErr
}

type managedHandler struct {
	slog.Handler
	state *lifecycleState
}

func (h *managedHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if h == nil {
		return nil
	}
	return &managedHandler{
		Handler: h.Handler.WithAttrs(attrs),
		state:   h.state,
	}
}

func (h *managedHandler) WithGroup(name string) slog.Handler {
	if h == nil {
		return nil
	}
	return &managedHandler{
		Handler: h.Handler.WithGroup(name),
		state:   h.state,
	}
}

func (h *managedHandler) Close() error {
	if h == nil {
		return nil
	}
	return h.state.close()
}

// New creates a slog.Logger using logx options.
func New(opts ...Option) (*slog.Logger, error) {
	cfg := defaultConfig()
	option.Apply(&cfg, opts...)

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	writers := collectionx.NewListWithCapacity[io.Writer](2)
	closers := collectionx.NewListWithCapacity[io.Closer](1)

	if cfg.console {
		writers.Add(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: cfg.timeFormat,
			NoColor:    cfg.noColor,
		})
	}

	if cfg.filePath != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.filePath), 0o750); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		fileWriter := &lumberjack.Logger{
			Filename:   cfg.filePath,
			MaxSize:    cfg.maxSize,
			MaxAge:     cfg.maxAge,
			MaxBackups: cfg.maxBackups,
			LocalTime:  cfg.localTime,
			Compress:   cfg.compress,
		}
		writers.Add(fileWriter)
		closers.Add(fileWriter)
	}

	if writers.Len() == 0 {
		writers.Add(os.Stdout)
	}

	oopsMarshalerOnce.Do(func() {
		zerolog.ErrorStackMarshaler = oopszerolog.OopsStackMarshaller
		zerolog.ErrorMarshalFunc = oopszerolog.OopsMarshalFunc
	})

	base := zerolog.New(io.MultiWriter(writers.Values()...)).
		Level(toZerologLevel(cfg.level)).
		With().
		Timestamp().
		Logger()

	if cfg.setGlobal {
		zlog.Logger = base
	}

	state := &lifecycleState{
		cfg:     cfg,
		closers: closers.Values(),
	}
	handler := slogzerolog.Option{
		Logger:    &base,
		AddSource: cfg.addCaller,
		Level:     cfg.level,
	}.NewZerologHandler()

	return slog.New(&managedHandler{
		Handler: handler,
		state:   state,
	}), nil
}

// MustNew creates a logger and panics on invalid configuration.
func MustNew(opts ...Option) *slog.Logger {
	logger, err := New(opts...)
	if err != nil {
		panic(err)
	}
	return logger
}

// NewDevelopment creates a development logger.
func NewDevelopment() (*slog.Logger, error) {
	return New(DevelopmentConfig().Values()...)
}

// NewProduction creates a production logger.
func NewProduction() (*slog.Logger, error) {
	return New(ProductionConfig("").Values()...)
}

// SetDefault sets slog default logger.
func SetDefault(logger *slog.Logger) *slog.Logger {
	if logger == nil {
		logger = slog.Default()
	}
	slog.SetDefault(logger)
	return logger
}

// Close closes resources associated with the logger.
func Close(logger *slog.Logger) error {
	if logger == nil {
		return nil
	}
	closer, ok := logger.Handler().(interface{ Close() error })
	if !ok {
		return nil
	}
	if err := closer.Close(); err != nil {
		return fmt.Errorf("close logger: %w", err)
	}
	return nil
}

// ConfigOf returns logger build config when logger was created by logx.New.
func ConfigOf(logger *slog.Logger) (Config, bool) {
	if logger == nil {
		return Config{}, false
	}

	handler, ok := logger.Handler().(*managedHandler)
	if !ok || handler.state == nil {
		return Config{}, false
	}
	return handler.state.cfg.export(), true
}

// WithField adds one field and returns a derived logger.
func WithField(logger *slog.Logger, key string, value any) *slog.Logger {
	if logger == nil {
		return nil
	}
	return logger.With(key, value)
}

// WithFields adds fields and returns a derived logger.
func WithFields(logger *slog.Logger, fields collectionx.Map[string, any]) *slog.Logger {
	if logger == nil {
		return nil
	}
	if fields == nil || fields.Len() == 0 {
		return logger
	}
	args := collectionx.NewListWithCapacity[any](fields.Len() * 2)
	fields.Range(func(key string, value any) bool {
		args.Add(key, value)
		return true
	})
	return logger.With(args.Values()...)
}

// WithError adds an error field and returns a derived logger.
func WithError(logger *slog.Logger, err error) *slog.Logger {
	if logger == nil {
		return nil
	}
	return logger.With("error", err)
}

// WithTraceContext adds trace and span IDs from context and returns a derived logger.
func WithTraceContext(ctx context.Context, logger *slog.Logger) *slog.Logger {
	if logger == nil || ctx == nil {
		return logger
	}

	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return logger
	}

	return logger.With(
		"trace_id", spanContext.TraceID().String(),
		"span_id", spanContext.SpanID().String(),
	)
}

// LevelOf returns configured level when logger was created by logx.New.
func LevelOf(logger *slog.Logger) (slog.Level, bool) {
	cfg, ok := ConfigOf(logger)
	if !ok {
		return slog.LevelInfo, false
	}
	return cfg.Level, true
}

// IsEnabled checks whether a level is enabled for current logger.
func IsEnabled(logger *slog.Logger, level slog.Level) bool {
	if logger == nil {
		return false
	}
	return logger.Enabled(context.Background(), level)
}
