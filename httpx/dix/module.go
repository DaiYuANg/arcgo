package httpxdix

import (
	"context"
	"log/slog"
	"strings"

	"github.com/DaiYuANg/arcgo/dix"
	"github.com/DaiYuANg/arcgo/httpx"
	"github.com/samber/oops"
)

type moduleOptions struct {
	imports         []dix.Module
	hooks           []dix.HookFunc
	moduleOptions   []dix.ModuleOption
	includeShutdown bool
}

// ModuleOption configures httpx/dix module construction.
type ModuleOption func(*moduleOptions)

// NewModule creates a dix module for an httpx server provider.
func NewModule(name string, provider dix.ProviderFunc, opts ...ModuleOption) dix.Module {
	cfg := moduleOptions{
		includeShutdown: true,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	moduleOpts := make([]dix.ModuleOption, 0, len(cfg.moduleOptions)+3)
	if len(cfg.imports) > 0 {
		moduleOpts = append(moduleOpts, dix.Imports(cfg.imports...))
	}
	moduleOpts = append(moduleOpts, dix.Providers(provider))

	hooks := make([]dix.HookFunc, 0, len(cfg.hooks)+1)
	if cfg.includeShutdown {
		hooks = append(hooks, Shutdown())
	}
	hooks = append(hooks, cfg.hooks...)
	if len(hooks) > 0 {
		moduleOpts = append(moduleOpts, dix.Hooks(hooks...))
	}

	moduleOpts = append(moduleOpts, cfg.moduleOptions...)
	return dix.NewModule(name, moduleOpts...)
}

// WithImports adds imported dix modules.
func WithImports(modules ...dix.Module) ModuleOption {
	return func(cfg *moduleOptions) {
		if cfg == nil {
			return
		}
		cfg.imports = append(cfg.imports, modules...)
	}
}

// WithHooks appends additional lifecycle hooks.
func WithHooks(hooks ...dix.HookFunc) ModuleOption {
	return func(cfg *moduleOptions) {
		if cfg == nil {
			return
		}
		cfg.hooks = append(cfg.hooks, hooks...)
	}
}

// WithModuleOptions appends raw dix module options.
func WithModuleOptions(opts ...dix.ModuleOption) ModuleOption {
	return func(cfg *moduleOptions) {
		if cfg == nil {
			return
		}
		cfg.moduleOptions = append(cfg.moduleOptions, opts...)
	}
}

// WithoutShutdown disables the default httpx shutdown hook.
func WithoutShutdown() ModuleOption {
	return func(cfg *moduleOptions) {
		if cfg == nil {
			return
		}
		cfg.includeShutdown = false
	}
}

// WithListen starts the server on a fixed address during app startup.
func WithListen(addr string) ModuleOption {
	return WithHooks(Listen(addr))
}

// WithListenPort starts the server on a fixed port during app startup.
func WithListenPort(port int) ModuleOption {
	return WithHooks(ListenPort(port))
}

// WithListen1 starts the server using an address resolved from one dix dependency.
func WithListen1[D1 any](resolver func(D1) string) ModuleOption {
	return WithHooks(Listen1(resolver))
}

// WithListenPort1 starts the server using a port resolved from one dix dependency.
func WithListenPort1[D1 any](resolver func(D1) int) ModuleOption {
	return WithHooks(ListenPort1(resolver))
}

// Shutdown stops the resolved httpx server during app shutdown.
func Shutdown() dix.HookFunc {
	return dix.OnStop(func(_ context.Context, server httpx.ServerRuntime) error {
		return server.Shutdown()
	})
}

// Listen starts the resolved httpx server on the provided address in the background.
func Listen(addr string) dix.HookFunc {
	return dix.OnStart(func(_ context.Context, server httpx.ServerRuntime) error {
		if strings.TrimSpace(addr) == "" {
			return oops.In("httpx/dix").
				With("op", "listen", "addr", addr).
				New("listen address is empty")
		}
		startBackground(server, "address", addr, func() error {
			return server.Listen(addr)
		})
		return nil
	})
}

// ListenPort starts the resolved httpx server on the provided port in the background.
func ListenPort(port int) dix.HookFunc {
	return dix.OnStart(func(_ context.Context, server httpx.ServerRuntime) error {
		if port < 0 || port > 65535 {
			return oops.In("httpx/dix").
				With("op", "listen_port", "port", port).
				Errorf("invalid port %d", port)
		}
		startBackground(server, "port", port, func() error {
			return server.ListenPort(port)
		})
		return nil
	})
}

// Listen1 starts the resolved httpx server using an address derived from one dependency.
func Listen1[D1 any](resolver func(D1) string) dix.HookFunc {
	return dix.OnStart2(func(_ context.Context, server httpx.ServerRuntime, dep D1) error {
		if resolver == nil {
			return oops.In("httpx/dix").
				With("op", "listen_dep").
				New("listen resolver is nil")
		}
		addr := resolver(dep)
		if strings.TrimSpace(addr) == "" {
			return oops.In("httpx/dix").
				With("op", "listen_dep", "addr", addr).
				New("listen address is empty")
		}
		startBackground(server, "address", addr, func() error {
			return server.Listen(addr)
		})
		return nil
	})
}

// ListenPort1 starts the resolved httpx server using a port derived from one dependency.
func ListenPort1[D1 any](resolver func(D1) int) dix.HookFunc {
	return dix.OnStart2(func(_ context.Context, server httpx.ServerRuntime, dep D1) error {
		if resolver == nil {
			return oops.In("httpx/dix").
				With("op", "listen_port_dep").
				New("listen port resolver is nil")
		}
		port := resolver(dep)
		if port < 0 || port > 65535 {
			return oops.In("httpx/dix").
				With("op", "listen_port_dep", "port", port).
				Errorf("invalid port %d", port)
		}
		startBackground(server, "port", port, func() error {
			return server.ListenPort(port)
		})
		return nil
	})
}

func startBackground(server httpx.ServerRuntime, key string, value any, listen func() error) {
	if server == nil || listen == nil {
		return
	}
	logger := server.Logger()
	if logger != nil {
		logger.Info("httpx/dix listen scheduled", key, value)
	}
	go func() {
		if err := listen(); err != nil {
			if logger != nil {
				logger.Error("httpx/dix server stopped", key, value, slog.String("error", err.Error()))
			}
		}
	}()
}
