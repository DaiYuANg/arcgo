package httpx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/DaiYuANg/toolkit4go/httpx/adapter"
	"github.com/DaiYuANg/toolkit4go/httpx/adapter/std"
	"github.com/samber/lo"
)

var (
	contextType         = reflect.TypeOf((*context.Context)(nil)).Elem()
	responseWriterType  = reflect.TypeOf((*http.ResponseWriter)(nil)).Elem()
	requestPtrType      = reflect.TypeOf(&http.Request{})
	errorType           = reflect.TypeOf((*error)(nil)).Elem()
	errHandlerNotFound  = errors.New("handler not found")
	errUnsupportedParam = errors.New("unsupported handler parameter")
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
		normalized := normalizeRoutePrefix(path)
		s.basePath = normalized
		s.generator.opts.BasePath = normalized
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
	method := reflect.ValueOf(s.adapter).MethodByName("WithHuma")
	if !method.IsValid() {
		return
	}

	methodType := method.Type()
	if methodType.NumIn() != 1 || methodType.In(0) != reflect.TypeOf(adapter.HumaOptions{}) {
		return
	}

	method.Call([]reflect.Value{reflect.ValueOf(s.humaOpts)})
}

// Register 注册 endpoint
func (s *Server) Register(endpoints ...interface{}) error {
	for _, endpoint := range endpoints {
		result := s.generator.Generate(endpoint)
		routes, err := result.Get()
		if err != nil {
			return err
		}

		for _, route := range routes {
			handler, err := s.buildHandler(endpoint, route)
			if err != nil {
				return err
			}
			s.adapter.Handle(route.Method, route.Path, handler)
			s.routes = append(s.routes, route)
		}
	}

	s.printRoutesIfEnabled()
	return nil
}

// RegisterWithPrefix 注册 endpoint 并添加路径前缀
func (s *Server) RegisterWithPrefix(prefix string, endpoints ...interface{}) error {
	opts := DefaultGeneratorOptions()
	opts.BasePath = joinRoutePath(s.basePath, prefix)
	gen := NewRouterGenerator(opts)

	for _, endpoint := range endpoints {
		result := gen.Generate(endpoint)
		routes, err := result.Get()
		if err != nil {
			return err
		}

		for _, route := range routes {
			handler, err := s.buildHandler(endpoint, route)
			if err != nil {
				return err
			}
			s.adapter.Handle(route.Method, route.Path, handler)
			s.routes = append(s.routes, route)
		}
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

// buildHandler 构建并校验 handler
func (s *Server) buildHandler(endpoint interface{}, route RouteInfo) (adapter.HandlerFunc, error) {
	v := reflect.ValueOf(endpoint)
	method := v.MethodByName(route.HandlerName)

	if !method.IsValid() {
		return nil, fmt.Errorf("%w: %s (%w)", ErrInvalidHandlerSig, route.HandlerName, errHandlerNotFound)
	}

	methodType := method.Type()
	if methodType.IsVariadic() {
		return nil, fmt.Errorf("%w: %s uses variadic params", ErrInvalidHandlerSig, route.HandlerName)
	}

	argKinds := make([]reflect.Type, methodType.NumIn())
	seenTypes := make(map[reflect.Type]struct{}, methodType.NumIn())
	for i := 0; i < methodType.NumIn(); i++ {
		paramType := methodType.In(i)
		switch paramType {
		case contextType, responseWriterType, requestPtrType:
			if _, exists := seenTypes[paramType]; exists {
				return nil, fmt.Errorf("%w: %s has duplicate param type %s", ErrInvalidHandlerSig, route.HandlerName, paramType.String())
			}
			seenTypes[paramType] = struct{}{}
			argKinds[i] = paramType
		default:
			return nil, fmt.Errorf("%w: %s param %d (%s): %w", ErrInvalidHandlerSig, route.HandlerName, i, paramType.String(), errUnsupportedParam)
		}
	}

	if methodType.NumOut() > 1 {
		return nil, fmt.Errorf("%w: %s must return zero values or one error", ErrInvalidHandlerSig, route.HandlerName)
	}
	returnsErr := methodType.NumOut() == 1
	if returnsErr && !methodType.Out(0).Implements(errorType) {
		return nil, fmt.Errorf("%w: %s return type must be error", ErrInvalidHandlerSig, route.HandlerName)
	}

	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) (callErr error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				callErr = fmt.Errorf("panic in handler %s: %v", route.HandlerName, recovered)
			}
		}()

		args := make([]reflect.Value, len(argKinds))
		for i, argType := range argKinds {
			switch argType {
			case contextType:
				args[i] = reflect.ValueOf(ctx)
			case responseWriterType:
				args[i] = reflect.ValueOf(w)
			case requestPtrType:
				args[i] = reflect.ValueOf(r)
			}
		}

		results := method.Call(args)
		if returnsErr && len(results) == 1 {
			if err, ok := results[0].Interface().(error); ok {
				return err
			}
		}
		return nil
	}, nil
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

	if s.isFiberAdapter() {
		return s.startFiberServer(addr)
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
		return fmt.Errorf("%w: fiber app does not implement Listen", ErrAdapterNotFound)
	}
	return fmt.Errorf("%w: fiber adapter does not expose App()", ErrAdapterNotFound)
}

func (s *Server) startFiberServerWithContext(ctx context.Context, addr string) error {
	adapt, ok := s.adapter.(interface{ App() interface{} })
	if !ok {
		return fmt.Errorf("%w: fiber adapter does not expose App()", ErrAdapterNotFound)
	}

	app, ok := adapt.App().(interface {
		Listen(string) error
		Shutdown() error
	})
	if !ok {
		return fmt.Errorf("%w: fiber app does not support graceful shutdown", ErrAdapterNotFound)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Listen(addr)
	}()

	select {
	case err := <-errCh:
		if isExpectedServerClose(err) {
			return nil
		}
		return err
	case <-ctx.Done():
		s.logger.Info("Shutting down fiber server")
		shutdownErr := app.Shutdown()
		listenErr := <-errCh
		if shutdownErr != nil {
			return shutdownErr
		}
		if isExpectedServerClose(listenErr) {
			return nil
		}
		return listenErr
	}
}

// ListenAndServeContext 启动服务器（支持 context）
func (s *Server) ListenAndServeContext(ctx context.Context, addr string) error {
	if s.isFiberAdapter() {
		return s.startFiberServerWithContext(ctx, addr)
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

func (s *Server) isFiberAdapter() bool {
	return s.adapter != nil && s.adapter.Name() == "fiber"
}

func isExpectedServerClose(err error) bool {
	if err == nil {
		return true
	}

	if errors.Is(err, http.ErrServerClosed) {
		return true
	}

	// Fiber 关闭时返回的错误类型来自 fiber 包，这里用消息兜底。
	return strings.Contains(strings.ToLower(err.Error()), "server is not running") ||
		strings.Contains(strings.ToLower(err.Error()), "use of closed network connection")
}
