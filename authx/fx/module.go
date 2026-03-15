package fx

import (
	"github.com/DaiYuANg/archgo/authx"
	"github.com/samber/lo"
	"go.uber.org/fx"
)

// EngineParams defines parameters for authx fx module.
type EngineParams struct {
	fx.In

	// Options grouped from WithEngineOptions and NewAuthxModule arguments.
	Options []authx.EngineOption `group:"authx_engine_options"`
}

// EngineResult defines result for authx fx module.
type EngineResult struct {
	fx.Out

	// Engine is the created authx engine.
	Engine *authx.Engine
}

// NewEngine creates an authx engine from grouped options.
func NewEngine(params EngineParams) EngineResult {
	return EngineResult{Engine: authx.NewEngine(params.Options...)}
}

// WithEngineOptions adds engine options into fx option group.
func WithEngineOptions(opts ...authx.EngineOption) fx.Option {
	filtered := lo.Filter(opts, func(item authx.EngineOption, _ int) bool {
		return item != nil
	})
	if len(filtered) == 0 {
		return fx.Options()
	}

	return fx.Provide(
		fx.Annotate(
			func() []authx.EngineOption { return filtered },
			fx.ResultTags(`group:"authx_engine_options,flatten"`),
		),
	)
}

// NewAuthxModule creates an authx fx module.
// It reuses authx.EngineOption as the module input options.
func NewAuthxModule(opts ...authx.EngineOption) fx.Option {
	return fx.Module("authx",
		fx.Provide(NewEngine),
		WithEngineOptions(opts...),
	)
}
