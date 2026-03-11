package httpx

import (
	"context"
	"log/slog"

	"github.com/danielgtaylor/huma/v2/humacli"
	"github.com/samber/mo"
)

// BindGracefulShutdownHooks wires the server lifecycle to Huma CLI hooks.
func BindGracefulShutdownHooks(hooks humacli.Hooks, server ServerRuntime, addr string) {
	if hooks == nil || server == nil || addr == "" {
		return
	}

	runCtx, cancel := context.WithCancel(context.Background())

	hooks.OnStart(func() {
		if err := server.ListenAndServeContext(runCtx, addr); err != nil {
			logger := mo.TupleToOption(server.Logger(), server.Logger() != nil).OrElse(slog.Default())
			logger.Error("httpx server exited", slog.String("address", addr), slog.String("error", err.Error()))
		}
	})

	hooks.OnStop(func() {
		cancel()
	})
}
