package std

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

// Adapter 标准 net/http 库适配器（基于 chi）
//
// 使用方式：
// 1. 创建适配器：stdAdapter := std.New()
// 2. 注册 chi 原生中间件：adapter.Router().Use(yourMiddleware...)
// 3. 创建 httpx server 并注册路由
type Adapter struct {
	router  *chi.Mux
	prefix  string
	logger  *slog.Logger
	huma    huma.API
	humaCfg adapter.HumaOptions
}

// New 创建标准 HTTP 适配器
func New() *Adapter {
	router := chi.NewMux()

	return &Adapter{
		router: router,
		logger: slog.Default(),
	}
}

// WithHuma 启用 Huma OpenAPI 文档
func (a *Adapter) WithHuma(opts adapter.HumaOptions) *Adapter {
	a.humaCfg = opts
	cfg := huma.DefaultConfig(opts.Title, opts.Version)
	cfg.OpenAPI.Info.Description = opts.Description
	a.huma = humachi.New(a.router, cfg)
	return a
}

// EnableHuma 启用 Huma OpenAPI 文档
func (a *Adapter) EnableHuma(opts adapter.HumaOptions) {
	a.WithHuma(opts)
}

// WithLogger 设置日志记录器
func (a *Adapter) WithLogger(logger *slog.Logger) *Adapter {
	a.logger = logger
	return a
}

// Name 返回适配器名称
func (a *Adapter) Name() string {
	return "std"
}

// Handle 注册业务处理函数
func (a *Adapter) Handle(method, path string, handler adapter.HandlerFunc) {
	fullPath := joinPath(a.prefix, path)
	a.router.Method(method, fullPath, a.wrapHandler(handler))
}

// Group 创建路由组
func (a *Adapter) Group(prefix string) adapter.Adapter {
	nextPrefix := a.prefix
	if prefix != "" && prefix != "/" {
		nextPrefix = joinPath(a.prefix, prefix)
	}
	return &Adapter{
		router:  a.router,
		prefix:  nextPrefix,
		logger:  a.logger,
		huma:    a.huma,
		humaCfg: a.humaCfg,
	}
}

// ServeHTTP 实现 http.Handler 接口
func (a *Adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}

// Router 返回底层 chi 路由器
// 通过此方法可以直接使用 chi 的中间件生态
// 例如：adapter.Router().Use(yourMiddleware...)
func (a *Adapter) Router() *chi.Mux {
	return a.router
}

// Listen 启动标准 HTTP 服务。
func (a *Adapter) Listen(addr string) error {
	return http.ListenAndServe(addr, a.router)
}

// ListenContext 启动标准 HTTP 服务并在 ctx 结束时优雅关闭。
func (a *Adapter) ListenContext(ctx context.Context, addr string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	server := &http.Server{
		Addr:    addr,
		Handler: a.router,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		err := <-errCh
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

// wrapHandler 包装处理函数
func (a *Adapter) wrapHandler(handler adapter.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := handler(r.Context(), w, r); err != nil {
			a.logger.Error("Handler error",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("error", err.Error()),
			)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// HumaAPI 返回 Huma API
func (a *Adapter) HumaAPI() huma.API {
	return a.huma
}

// HasHuma 检查是否启用了 Huma
func (a *Adapter) HasHuma() bool {
	return a.huma != nil
}

func joinPath(prefix, path string) string {
	cleanPrefix := strings.TrimRight(prefix, "/")
	if cleanPrefix == "" {
		if path == "" {
			return "/"
		}
		if strings.HasPrefix(path, "/") {
			return path
		}
		return "/" + path
	}

	if path == "" || path == "/" {
		return cleanPrefix
	}
	if strings.HasPrefix(path, "/") {
		return cleanPrefix + path
	}
	return cleanPrefix + "/" + path
}
