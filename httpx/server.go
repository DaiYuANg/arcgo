package httpx

import (
	"context"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/DaiYuANg/toolkit4go/httpx/adapter"
	"github.com/DaiYuANg/toolkit4go/httpx/adapter/std"
	"github.com/samber/lo"
)

// Server HTTP 服务器
//
// 设计理念：
// 1. httpx 核心职责：减少样板代码，方便路由注册和管理
// 2. 中间件：直接使用各框架的原生方式注册
// 3. 适配器：提供底层框架的原生访问接口（如 Engine()、App()、Router()）
//
// 使用示例（Gin）：
//
//	// 创建适配器
//	ginAdapter := gin.New()
//	// 使用 Gin 原生方式注册中间件
//	ginAdapter.Engine().Use(gin.Logger(), yourMiddleware...)
//	// 创建 server 并注册业务路由
//	server := httpx.NewServer(httpx.WithAdapter(ginAdapter))
//	server.Register(&YourEndpoint{})
type Server struct {
	adapter     adapter.Adapter
	generator   *RouterGenerator
	basePath    string
	routes      []RouteInfo
	logger      *slog.Logger
	printRoutes bool
	humaOpts    adapter.HumaOptions
}

// ServerOption Server 配置选项
type ServerOption func(*Server)

// WithAdapter 设置适配器
func WithAdapter(adapter adapter.Adapter) ServerOption {
	return func(s *Server) {
		s.adapter = adapter
	}
}

// WithAdapterName 通过名称设置适配器（已废弃，请使用 WithAdapter）
// Deprecated: 请直接使用各框架的 adapter 子包，如 adapter/gin.New()
func WithAdapterName(name string) ServerOption {
	return func(s *Server) {
		s.logger.Warn("WithAdapterName is deprecated, use adapter subpackages directly")
	}
}

// WithGenerator 设置路由生成器
func WithGenerator(gen *RouterGenerator) ServerOption {
	return func(s *Server) {
		s.generator = gen
	}
}

// WithBasePath 设置基础路径
func WithBasePath(path string) ServerOption {
	return func(s *Server) {
		s.basePath = path
		s.generator.opts.BasePath = path
	}
}

// WithLogger 设置日志记录器
func WithLogger(logger *slog.Logger) ServerOption {
	return func(s *Server) {
		s.logger = logger
	}
}

// WithPrintRoutes 设置是否打印路由
func WithPrintRoutes(enabled bool) ServerOption {
	return func(s *Server) {
		s.printRoutes = enabled
	}
}

// WithHuma 配置 Huma OpenAPI 文档
func WithHuma(opts HumaOptions) ServerOption {
	return func(s *Server) {
		s.humaOpts = adapter.HumaOptions(opts)
	}
}

// NewServer 创建 HTTP 服务器
func NewServer(opts ...ServerOption) *Server {
	s := &Server{
		generator: NewRouterGenerator(),
		logger:    slog.Default(),
		routes:    make([]RouteInfo, 0),
	}

	lo.ForEach(opts, func(opt ServerOption, index int) {
		opt(s)
	})

	if s.adapter == nil {
		// 默认使用 std adapter
		s.adapter = std.New()
	}

	// 如果启用了 Huma，配置适配器
	if s.humaOpts.Enabled {
		s.configureAdapterHuma()
	}

	return s
}

// configureAdapterHuma 配置适配器的 Huma 支持
func (s *Server) configureAdapterHuma() {
	if configurable, ok := s.adapter.(interface {
		WithHuma(adapter.HumaOptions) adapter.Adapter
	}); ok {
		configurable.WithHuma(s.humaOpts)
	}
}

// Register 注册 endpoint
func (s *Server) Register(endpoints ...interface{}) error {
	for _, endpoint := range endpoints {
		result := s.generator.Generate(endpoint)
		routes, err := result.Get()
		if err != nil {
			return err
		}

		lo.ForEach(routes, func(route RouteInfo, _ int) {
			s.adapter.Handle(route.Method, route.Path, s.wrapHandler(endpoint, route))
			s.routes = append(s.routes, route)
		})
	}

	s.printRoutesIfEnabled()
	return nil
}

// RegisterWithPrefix 注册 endpoint 并添加路径前缀
func (s *Server) RegisterWithPrefix(prefix string, endpoints ...interface{}) error {
	opts := DefaultGeneratorOptions()
	opts.BasePath = s.basePath + prefix
	gen := NewRouterGenerator(opts)

	for _, endpoint := range endpoints {
		result := gen.Generate(endpoint)
		routes, err := result.Get()
		if err != nil {
			return err
		}

		lo.ForEach(routes, func(route RouteInfo, _ int) {
			s.adapter.Handle(route.Method, route.Path, s.wrapHandler(endpoint, route))
			s.routes = append(s.routes, route)
		})
	}

	s.printRoutesIfEnabled()
	return nil
}

// printRoutesIfEnabled 打印路由
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

// GetRoutes 返回所有路由
func (s *Server) GetRoutes() []RouteInfo {
	return lo.Map(s.routes, func(route RouteInfo, _ int) RouteInfo {
		return route
	})
}

// GetRoutesByMethod 按方法过滤路由
func (s *Server) GetRoutesByMethod(method string) []RouteInfo {
	return lo.Filter(s.routes, func(route RouteInfo, _ int) bool {
		return route.Method == method
	})
}

// GetRoutesByPath 按路径过滤路由
func (s *Server) GetRoutesByPath(prefix string) []RouteInfo {
	return lo.Filter(s.routes, func(route RouteInfo, _ int) bool {
		return len(prefix) == 0 || strings.HasPrefix(route.Path, prefix)
	})
}

// HasRoute 检查路由是否存在
func (s *Server) HasRoute(method, path string) bool {
	return lo.SomeBy(s.routes, func(route RouteInfo) bool {
		return route.Method == method && route.Path == path
	})
}

// RouteCount 返回路由数量
func (s *Server) RouteCount() int {
	return len(s.routes)
}

// wrapHandler 包装 handler
func (s *Server) wrapHandler(endpoint interface{}, route RouteInfo) adapter.HandlerFunc {
	v := reflect.ValueOf(endpoint)
	method := v.MethodByName(route.HandlerName)

	if !method.IsValid() {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			return NewError(http.StatusInternalServerError, "handler not found")
		}
	}

	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		methodType := method.Type()
		args := make([]reflect.Value, methodType.NumIn())

		for i := 0; i < methodType.NumIn(); i++ {
			paramType := methodType.In(i)
			if paramType.Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
				args[i] = reflect.ValueOf(ctx)
			}
			if paramType.Implements(reflect.TypeOf((*http.ResponseWriter)(nil)).Elem()) {
				args[i] = reflect.ValueOf(w)
			}
			if paramType == reflect.TypeOf(&http.Request{}) {
				args[i] = reflect.ValueOf(r)
			}
		}

		results := method.Call(args)
		if len(results) > 0 {
			if err, ok := results[0].Interface().(error); ok && err != nil {
				return err
			}
		}
		return nil
	}
}

// Handler 返回 http.Handler
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.adapter.ServeHTTP(w, r)
	})
}

// ServeHTTP 实现 http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Handler().ServeHTTP(w, r)
}

// ListenAndServe 启动服务器
func (s *Server) ListenAndServe(addr string) error {
	s.logger.Info("Starting server",
		slog.String("address", addr),
		slog.String("adapter", s.adapter.Name()),
		slog.Int("routes", len(s.routes)),
	)

	// 如果启用了 Huma，需要组合路由（Fiber 除外）
	if s.HasHuma() {
		// 检查是否是 Fiber adapter（通过名称判断）
		if s.adapter.Name() == "fiber" {
			return s.startFiberServer(addr)
		}

		mux := http.NewServeMux()

		// 注册应用路由
		mux.Handle("/", s.adapter)

		return http.ListenAndServe(addr, mux)
	}

	return http.ListenAndServe(addr, s.Handler())
}

// startFiberServer 启动 Fiber 服务器
func (s *Server) startFiberServer(addr string) error {
	// Fiber adapter 有自己的 Listen 方法
	// 通过类型断言获取 App 并调用 Listen
	if adapter, ok := s.adapter.(interface{ App() interface{} }); ok {
		if app, ok := adapter.App().(interface{ Listen(string) error }); ok {
			return app.Listen(addr)
		}
	}
	return nil
}

// ListenAndServeContext 启动服务器（支持 context）
func (s *Server) ListenAndServeContext(ctx context.Context, addr string) error {
	server := &http.Server{
		Addr:    addr,
		Handler: s.Handler(),
	}

	s.logger.Info("Starting server with context", slog.String("address", addr))

	go func() {
		<-ctx.Done()
		s.logger.Info("Shutting down server")
		server.Shutdown(context.Background())
	}()

	return server.ListenAndServe()
}

// Adapter 返回适配器
func (s *Server) Adapter() adapter.Adapter {
	return s.adapter
}

// Logger 返回日志记录器
func (s *Server) Logger() *slog.Logger {
	return s.logger
}

// HasHuma 检查是否启用了 Huma
func (s *Server) HasHuma() bool {
	if hasHuma, ok := s.adapter.(interface{ HasHuma() bool }); ok {
		return hasHuma.HasHuma()
	}
	return false
}
