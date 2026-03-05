package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/DaiYuANg/arcgo/httpx/adapter/std"
	"github.com/danielgtaylor/huma/v2"
	"github.com/samber/lo"
)

// Server HTTP 服务器。
//
// 设计说明：
// 1. 以泛型强类型路由注册为核心（见 Route/Get/Post 等泛型函数）
// 2. Huma 是可选能力：启用后使用 huma.Register；关闭时走内置 JSON 编解码
// 3. 中间件始终建议通过各框架原生方式注册
type Server struct {
	adapter     adapter.Adapter
	basePath    string
	routes      []RouteInfo
	logger      *slog.Logger
	printRoutes bool
	humaOpts    adapter.HumaOptions
}

// Group 泛型路由组。
type Group struct {
	server *Server
	prefix string
}

// ServerOption Server 配置选项。
type ServerOption func(*Server)

// WithAdapter 设置适配器。
func WithAdapter(adapter adapter.Adapter) ServerOption {
	return func(s *Server) {
		s.adapter = adapter
	}
}

// WithAdapterName 通过名称设置适配器（已废弃，请使用 WithAdapter）。
// Deprecated: 请直接使用各框架的 adapter 子包，如 adapter/gin.New()
func WithAdapterName(name string) ServerOption {
	return func(s *Server) {
		s.logger.Warn("WithAdapterName is deprecated, use adapter subpackages directly")
	}
}

// WithBasePath 设置基础路径。
func WithBasePath(path string) ServerOption {
	return func(s *Server) {
		s.basePath = normalizeRoutePrefix(path)
	}
}

// WithLogger 设置日志记录器。
func WithLogger(logger *slog.Logger) ServerOption {
	return func(s *Server) {
		s.logger = logger
	}
}

// WithPrintRoutes 设置是否打印路由。
func WithPrintRoutes(enabled bool) ServerOption {
	return func(s *Server) {
		s.printRoutes = enabled
	}
}

// WithHuma 配置 Huma OpenAPI 文档。
func WithHuma(opts HumaOptions) ServerOption {
	return func(s *Server) {
		s.humaOpts = adapter.HumaOptions(opts)
	}
}

// NewServer 创建 HTTP 服务器。
func NewServer(opts ...ServerOption) *Server {
	s := &Server{
		logger:   slog.Default(),
		routes:   make([]RouteInfo, 0),
		humaOpts: adapter.DefaultHumaOptions(),
	}

	lo.ForEach(opts, func(opt ServerOption, _ int) {
		opt(s)
	})

	if s.adapter == nil {
		// 默认使用 std adapter
		s.adapter = std.New()
	}

	if s.humaOpts.Enabled {
		s.adapter.EnableHuma(s.humaOpts)
	}

	return s
}

// Group 创建路由分组。
func (s *Server) Group(prefix string) *Group {
	return &Group{
		server: s,
		prefix: normalizeRoutePrefix(prefix),
	}
}

func (s *Server) writeHandlerError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

	status := http.StatusInternalServerError
	message := err.Error()

	var httpxErr *Error
	if errors.As(err, &httpxErr) {
		status = httpxErr.Code
		message = httpxErr.Message
	}

	s.logger.Error("Handler error",
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Int("status", status),
		slog.String("error", err.Error()),
	)

	writeJSON(w, status, map[string]any{
		"error": message,
	})
}

// printRoutesIfEnabled 打印路由。
func (s *Server) printRoutesIfEnabled() {
	if !s.printRoutes {
		return
	}

	s.logger.Info("Registered routes", slog.Int("count", len(s.routes)))
	lo.ForEach(s.routes, func(route RouteInfo, _ int) {
		s.logger.Info("  "+route.String(),
			slog.String("method", route.Method),
			slog.String("path", route.Path),
			slog.String("handler", route.HandlerName),
		)
	})
}

// GetRoutes 返回所有路由。
func (s *Server) GetRoutes() []RouteInfo {
	return lo.Map(s.routes, func(route RouteInfo, _ int) RouteInfo {
		return route
	})
}

// GetRoutesByMethod 按方法过滤路由。
func (s *Server) GetRoutesByMethod(method string) []RouteInfo {
	return lo.Filter(s.routes, func(route RouteInfo, _ int) bool {
		return route.Method == strings.ToUpper(method)
	})
}

// GetRoutesByPath 按路径过滤路由。
func (s *Server) GetRoutesByPath(prefix string) []RouteInfo {
	return lo.Filter(s.routes, func(route RouteInfo, _ int) bool {
		return len(prefix) == 0 || strings.HasPrefix(route.Path, prefix)
	})
}

// HasRoute 检查路由是否存在。
func (s *Server) HasRoute(method, path string) bool {
	upperMethod := strings.ToUpper(method)
	return lo.SomeBy(s.routes, func(route RouteInfo) bool {
		return route.Method == upperMethod && route.Path == path
	})
}

// RouteCount 返回路由数量。
func (s *Server) RouteCount() int {
	return len(s.routes)
}

// Handler 返回 http.Handler。
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.adapter.ServeHTTP(w, r)
	})
}

// ServeHTTP 实现 http.Handler。
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Handler().ServeHTTP(w, r)
}

// ListenAndServe 启动服务器。
func (s *Server) ListenAndServe(addr string) error {
	s.logger.Info("Starting server",
		slog.String("address", addr),
		slog.String("adapter", s.adapter.Name()),
		slog.Int("routes", len(s.routes)),
		slog.Bool("huma_enabled", s.humaOpts.Enabled),
	)

	if listenable, ok := s.adapter.(adapter.ListenableAdapter); ok {
		return listenable.Listen(addr)
	}

	return http.ListenAndServe(addr, s.Handler())
}

// ListenAndServeContext 启动服务器（支持 context）。
func (s *Server) ListenAndServeContext(ctx context.Context, addr string) error {
	if listenable, ok := s.adapter.(adapter.ContextListenableAdapter); ok {
		return listenable.ListenContext(ctx, addr)
	}

	server := &http.Server{
		Addr:    addr,
		Handler: s.Handler(),
	}

	s.logger.Info("Starting server with context", slog.String("address", addr))

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
		s.logger.Info("Shutting down server")
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

// Adapter 返回适配器。
func (s *Server) Adapter() adapter.Adapter {
	return s.adapter
}

// Logger 返回日志记录器。
func (s *Server) Logger() *slog.Logger {
	return s.logger
}

// HumaAPI 返回 Huma API（未启用时为 nil）。
func (s *Server) HumaAPI() huma.API {
	return s.adapter.HumaAPI()
}

// HasHuma 检查是否启用了 Huma。
func (s *Server) HasHuma() bool {
	return s.adapter.HasHuma()
}
