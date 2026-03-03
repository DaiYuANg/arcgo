package logx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"github.com/samber/lo"
	oopszerolog "github.com/samber/oops/loggers/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger 日志记录器（基于 zerolog）
type Logger struct {
	logger  zerolog.Logger
	closers []io.Closer
	config  *config
}

// Close 关闭日志记录器，释放文件句柄
func (l *Logger) Close() error {
	errs := lo.FilterMap(l.closers, func(closer io.Closer, _ int) (error, bool) {
		if closer == nil {
			return nil, false
		}
		err := closer.Close()
		return err, err != nil
	})
	return errors.Join(errs...)
}

// Config 返回当前配置
func (l *Logger) Config() *config {
	return l.config
}

// New 创建日志记录器
// 支持多个 Option 配置
// 返回 *Logger 和 error，如果配置无效返回 error
func New(opts ...Option) (*Logger, error) {
	cfg := defaultConfig()

	for _, opt := range opts {
		opt(&cfg)
	}

	// 验证配置
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	var writers []io.Writer
	var closers []io.Closer

	// console
	if cfg.console {
		writers = append(writers, zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: cfg.timeFormat,
			NoColor:    cfg.noColor,
		})
	}

	// file
	if cfg.filePath != "" {
		// 确保目录存在
		if err := os.MkdirAll(filepath.Dir(cfg.filePath), 0o755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		lj := &lumberjack.Logger{
			Filename:   cfg.filePath,
			MaxSize:    cfg.maxSize,
			MaxAge:     cfg.maxAge,
			MaxBackups: cfg.maxBackups,
			LocalTime:  cfg.localTime,
			Compress:   cfg.compress,
		}
		writers = append(writers, lj)
		closers = append(closers, lj)
	}

	// 如果没有指定输出，默认输出到 stdout
	if len(writers) == 0 {
		writers = append(writers, os.Stdout)
	}

	level := cfg.level.ToZerologLevel()

	mw := io.MultiWriter(writers...)

	// 创建 zerolog logger
	builder := zerolog.New(mw).
		Level(level).
		With().
		Timestamp()

	if cfg.addCaller {
		builder = builder.Caller()
	}

	z := builder.Logger()

	// 设置 oops 的 zerolog 集成
	zerolog.ErrorStackMarshaler = oopszerolog.OopsStackMarshaller
	zerolog.ErrorMarshalFunc = oopszerolog.OopsMarshalFunc

	// 设置为全局 logger
	if cfg.setGlobal {
		zlog.Logger = z
	}

	return &Logger{
		logger:  z,
		closers: closers,
		config:  &cfg,
	}, nil
}

// MustNew 创建日志记录器，如果失败则 panic
func MustNew(opts ...Option) *Logger {
	logger, err := New(opts...)
	if err != nil {
		panic(err)
	}
	return logger
}

// NewDevelopment 创建开发环境日志记录器
// 自动启用 console 输出和 debug 级别
func NewDevelopment() (*Logger, error) {
	return New(
		WithConsole(true),
		WithLevel(DebugLevel),
		WithCaller(true),
	)
}

// NewProduction 创建生产环境日志记录器
// 默认 JSON 格式，info 级别
func NewProduction() (*Logger, error) {
	return New(
		WithConsole(false),
		WithLevel(InfoLevel),
	)
}

// SetGlobalLogger 将当前 logger 设置为全局默认 logger
func (l *Logger) SetGlobalLogger() {
	zlog.Logger = l.logger
}

// WithContext 添加 context 到 logger
func (l *Logger) WithContext(ctx context.Context) zerolog.Context {
	return l.logger.With().Ctx(ctx)
}

// Logger 返回 zerolog.Logger
func (l *Logger) Logger() zerolog.Logger {
	return l.logger
}

// 便捷方法

// Debug 记录 debug 级别日志
func (l *Logger) Debug(msg string, fields ...interface{}) {
	l.logger.Debug().Fields(fields).Msg(msg)
}

// Info 记录 info 级别日志
func (l *Logger) Info(msg string, fields ...interface{}) {
	l.logger.Info().Fields(fields).Msg(msg)
}

// Warn 记录 warn 级别日志
func (l *Logger) Warn(msg string, fields ...interface{}) {
	l.logger.Warn().Fields(fields).Msg(msg)
}

// Error 记录 error 级别日志
func (l *Logger) Error(msg string, fields ...interface{}) {
	l.logger.Error().Fields(fields).Msg(msg)
}

// Fatal 记录 fatal 级别日志
func (l *Logger) Fatal(msg string, fields ...interface{}) {
	l.logger.Fatal().Fields(fields).Msg(msg)
}

// Panic 记录 panic 级别日志
func (l *Logger) Panic(msg string, fields ...interface{}) {
	l.logger.Panic().Fields(fields).Msg(msg)
}

// WithField 添加字段到 logger
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{
		logger:  l.logger.With().Interface(key, value).Logger(),
		closers: l.closers,
		config:  l.config,
	}
}

// WithFields 添加多个字段到 logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	logger := l.logger
	for k, v := range fields {
		logger = logger.With().Interface(k, v).Logger()
	}
	return &Logger{
		logger:  logger,
		closers: l.closers,
		config:  l.config,
	}
}

// WithError 添加 error 到 logger
func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		logger:  l.logger.With().Err(err).Logger(),
		closers: l.closers,
		config:  l.config,
	}
}

// WithCaller 启用/禁用调用者信息
func (l *Logger) WithCaller(enabled bool) *Logger {
	if enabled {
		return &Logger{
			logger:  l.logger.With().Caller().Logger(),
			closers: l.closers,
			config:  l.config,
		}
	}
	return &Logger{
		logger:  l.logger,
		closers: l.closers,
		config:  l.config,
	}
}

// Helper 标记为辅助函数（用于跳过调用栈）
func Helper() {
	_, file, line, _ := runtime.Caller(1)
	fmt.Printf("Called from %s:%d\n", file, line)
}

// Sync 同步日志缓冲区到磁盘
func (l *Logger) Sync() error {
	// zerolog 是同步写入的，不需要额外同步
	return nil
}

// GetLevel 获取当前日志级别（强类型）
func (l *Logger) GetLevel() Level {
	return l.config.level
}

// GetLevelString 获取当前日志级别的字符串表示
func (l *Logger) GetLevelString() string {
	return l.config.level.String()
}

// IsDebug 检查是否启用 debug 级别
func (l *Logger) IsDebug() bool {
	return l.config.level <= DebugLevel
}

// IsTrace 检查是否启用 trace 级别
func (l *Logger) IsTrace() bool {
	return l.config.level <= TraceLevel
}

// IsInfo 检查是否启用 info 级别
func (l *Logger) IsInfo() bool {
	return l.config.level <= InfoLevel
}

// IsWarn 检查是否启用 warn 级别
func (l *Logger) IsWarn() bool {
	return l.config.level <= WarnLevel
}

// IsError 检查是否启用 error 级别
func (l *Logger) IsError() bool {
	return l.config.level <= ErrorLevel
}
