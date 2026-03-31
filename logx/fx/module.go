package fx

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/DaiYuANg/arcgo/logx"
	pkgfx "github.com/DaiYuANg/arcgo/pkg/fx"
	"go.uber.org/fx"
)

// LogParams defines parameters for logx module.
type LogParams struct {
	fx.In

	Lifecycle fx.Lifecycle

	// Options for creating logger.
	Options []logx.Option `group:"logx_options,soft"`
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
		return LogResult{}, fmt.Errorf("create logx logger: %w", err)
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
		pkgfx.ProvideOptionGroup[logx.BuildOptions, logx.Option]("logx_options", opts...),
		fx.Provide(NewLogger),
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
