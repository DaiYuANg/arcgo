package fx

import (
	"context"
	"log/slog"

	"go.uber.org/fx"

	"github.com/DaiYuANg/arcgo/logx"
)

// LogParams defines parameters for logx module.
type LogParams struct {
	fx.In

	Lifecycle fx.Lifecycle

	// Options for creating logger.
	Options []logx.Option `optional:"true"`
}

// LogResult defines result for logx module.
type LogResult struct {
	fx.Out

	// Logger is the created logger.
	Logger *slog.Logger
}

// NewLogger creates a new logger.
func NewLogger(params LogParams) (LogResult, error) {
	logger, err := logx.New(params.Options...)
	if err != nil {
		return LogResult{}, err
	}
	params.Lifecycle.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return logx.Close(logger)
		},
	})
	return LogResult{Logger: logger}, nil
}

// NewLogxModule creates a logx module.
func NewLogxModule(opts ...logx.Option) fx.Option {
	return fx.Module("logx",
		fx.Provide(
			func() []logx.Option { return opts },
			NewLogger,
		),
	)
}

// NewLogxModuleWithSlog keeps parity with previous API; logger is slog-first by default.
func NewLogxModuleWithSlog(opts ...logx.Option) fx.Option {
	return NewLogxModule(opts...)
}

// NewDevelopmentModule creates a development logx module.
func NewDevelopmentModule() fx.Option {
	return NewLogxModule(logx.DevelopmentConfig()...)
}

// NewProductionModule creates a production logx module.
func NewProductionModule(logPath string) fx.Option {
	return NewLogxModule(logx.ProductionConfig(logPath)...)
}
