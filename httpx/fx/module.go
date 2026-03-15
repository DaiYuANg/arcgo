package fx

import (
	"github.com/DaiYuANg/archgo/httpx"
	"github.com/samber/lo"
	"go.uber.org/fx"
)

// ServerParams defines parameters for httpx fx module.
type ServerParams struct {
	fx.In

	// Options grouped from WithServerOptions and NewHttpxModule arguments.
	Options []httpx.ServerOption `group:"httpx_server_options"`
}

// ServerResult defines result for httpx fx module.
type ServerResult struct {
	fx.Out

	// Server is the created httpx server runtime.
	Server httpx.ServerRuntime
}

// NewServer creates a httpx server runtime from grouped options.
func NewServer(params ServerParams) ServerResult {
	return ServerResult{Server: httpx.New(params.Options...)}
}

// WithServerOptions adds server options into fx option group.
func WithServerOptions(opts ...httpx.ServerOption) fx.Option {
	filtered := lo.Filter(opts, func(item httpx.ServerOption, _ int) bool {
		return item != nil
	})
	if len(filtered) == 0 {
		return fx.Options()
	}

	return fx.Provide(
		fx.Annotate(
			func() []httpx.ServerOption { return filtered },
			fx.ResultTags(`group:"httpx_server_options,flatten"`),
		),
	)
}

// NewHttpxModule creates a httpx fx module.
// It reuses httpx.ServerOption as the module input options.
func NewHttpxModule(opts ...httpx.ServerOption) fx.Option {
	return fx.Module("httpx",
		fx.Provide(NewServer),
		WithServerOptions(opts...),
	)
}
