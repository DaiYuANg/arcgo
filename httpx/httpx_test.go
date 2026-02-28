package httpx

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DaiYuANg/toolkit4go/httpx/adapter/std"
	"github.com/stretchr/testify/assert"
)

// TestUserEndpoint 测试用的 endpoint
type TestUserEndpoint struct {
	BaseEndpoint
}

// GetUserList 获取用户列表 - 通过方法名自动生成 GET /user/list
func (e *TestUserEndpoint) GetUserList() {
}

// GetUserByID 获取单个用户 - 自动生成 GET /user/by/id
func (e *TestUserEndpoint) GetUserByID() {
}

// CreateUser 创建用户 - 自动生成 POST /user
func (e *TestUserEndpoint) CreateUser() {
}

// UpdateUser 更新用户 - 自动生成 PUT /user
func (e *TestUserEndpoint) UpdateUser() {
}

// DeleteUser 删除用户 - 自动生成 DELETE /user
func (e *TestUserEndpoint) DeleteUser() {
}

// TestRouterGenerator_Naming 测试方法名解析
func TestRouterGenerator_Naming(t *testing.T) {
	gen := NewRouterGenerator()
	endpoint := &TestUserEndpoint{}

	routes, err := gen.Generate(endpoint).Get()
	assert.NoError(t, err)
	// 包含 BaseEndpoint 的 GetHeader 和 GetQuery 方法
	assert.GreaterOrEqual(t, len(routes), 5)

	// 验证路由
	routeMap := make(map[string]RouteInfo)
	for _, route := range routes {
		routeMap[route.HandlerName] = route
	}

	// GetUserList -> GET /user/list
	assert.Equal(t, MethodGet, routeMap["GetUserList"].Method)
	assert.Equal(t, "/user/list", routeMap["GetUserList"].Path)

	// CreateUser -> POST /user (更 RESTful)
	assert.Equal(t, MethodPost, routeMap["CreateUser"].Method)
	assert.Equal(t, "/user", routeMap["CreateUser"].Path)

	// DeleteUser -> DELETE /user (更 RESTful)
	assert.Equal(t, MethodDelete, routeMap["DeleteUser"].Method)
	assert.Equal(t, "/user", routeMap["DeleteUser"].Path)
}

// TestTagEndpoint 使用标签的 endpoint
type TestTagEndpoint struct {
	GetUsers       func() `http:"GET /api/users"`
	GetUser        func() `http:"/api/user/:id"`
	CreateUser     func() `route:"POST /api/users"`
	UpdateUserInfo func() `route:"PUT /api/user/:id"`
}

// TestRouterGenerator_Tag 测试标签解析
func TestRouterGenerator_Tag(t *testing.T) {
	gen := NewRouterGenerator()
	endpoint := &TestTagEndpoint{}

	routes, err := gen.Generate(endpoint).Get()
	assert.NoError(t, err)
	assert.Len(t, routes, 4)

	routeMap := make(map[string]RouteInfo)
	for _, route := range routes {
		routeMap[route.HandlerName] = route
	}

	// 验证 http 标签
	assert.Equal(t, MethodGet, routeMap["GetUsers"].Method)
	assert.Equal(t, "/api/users", routeMap["GetUsers"].Path)

	assert.Equal(t, MethodGet, routeMap["GetUser"].Method)
	assert.Equal(t, "/api/user/:id", routeMap["GetUser"].Path)

	// 验证 route 标签
	assert.Equal(t, MethodPost, routeMap["CreateUser"].Method)
	assert.Equal(t, "/api/users", routeMap["CreateUser"].Path)

	assert.Equal(t, MethodPut, routeMap["UpdateUserInfo"].Method)
	assert.Equal(t, "/api/user/:id", routeMap["UpdateUserInfo"].Path)
}

// TestServerEndpoint 测试用的 handler endpoint
type TestServerEndpoint struct {
	BaseEndpoint
}

// GetItems 获取列表
func (e *TestServerEndpoint) GetItems(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	e.Success(w, map[string]interface{}{
		"items": []string{"item1", "item2", "item3"},
	})
	return nil
}

// CreateItem 创建
func (e *TestServerEndpoint) CreateItem(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	e.Success(w, map[string]interface{}{
		"message": "created",
	})
	return nil
}

// TestServer_StdHTTP 测试标准 HTTP 服务器
func TestServer_StdHTTP(t *testing.T) {
	// 创建实际的 handler endpoint
	endpoint := &TestServerEndpoint{}

	// 创建服务器
	server := NewServer()

	// 注册 endpoint
	err := server.Register(endpoint)
	assert.NoError(t, err)

	// 创建测试请求
	req := httptest.NewRequest(MethodGet, "/items", nil)
	w := httptest.NewRecorder()

	// 执行请求
	server.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "success", response["status"])
}

// TestServer_WithMiddleware 测试中间件
// 注意：中间件应该直接使用框架原生方式注册
func TestServer_WithMiddleware(t *testing.T) {
	var middlewareCalled bool

	// 使用 std adapter 直接在 router 上注册中间件
	stdAdapter := std.New()
	stdAdapter.Router().Use(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			handler.ServeHTTP(w, r)
		})
	})

	endpoint := &TestServerEndpoint{}
	server := NewServer(WithAdapter(stdAdapter))

	err := server.Register(endpoint)
	assert.NoError(t, err)

	req := httptest.NewRequest(MethodGet, "/items", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	assert.True(t, middlewareCalled, "middleware should be called")
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestServer_BasePath 测试基础路径
func TestServer_BasePath(t *testing.T) {
	endpoint := &TestServerEndpoint{}
	server := NewServer(WithBasePath("/api/v1"))

	err := server.Register(endpoint)
	assert.NoError(t, err)

	// 验证路由是否包含基础路径
	req := httptest.NewRequest(MethodGet, "/api/v1/items", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestServer_RegisterWithPrefix 测试带前缀的注册
func TestServer_RegisterWithPrefix(t *testing.T) {
	endpoint := &TestServerEndpoint{}
	server := NewServer()

	err := server.RegisterWithPrefix("/api/v1", endpoint)
	assert.NoError(t, err)

	req := httptest.NewRequest(MethodGet, "/api/v1/items", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestServer_WithLogger 测试自定义 Logger
func TestServer_WithLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(httptest.NewRecorder(), &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	endpoint := &TestServerEndpoint{}
	server := NewServer(WithLogger(logger))

	err := server.Register(endpoint)
	assert.NoError(t, err)

	assert.Equal(t, logger, server.Logger())
}

// TestServer_PrintRoutes 测试打印路由
func TestServer_PrintRoutes(t *testing.T) {
	endpoint := &TestServerEndpoint{}
	server := NewServer(WithPrintRoutes(true))

	err := server.Register(endpoint)
	assert.NoError(t, err)

	// 验证路由已保存
	assert.Greater(t, server.RouteCount(), 0)
	routes := server.GetRoutes()
	assert.NotEmpty(t, routes)
}

// TestServer_GetRoutes 测试获取路由
func TestServer_GetRoutes(t *testing.T) {
	endpoint := &TestServerEndpoint{}
	server := NewServer()

	err := server.Register(endpoint)
	assert.NoError(t, err)

	// 测试 GetRoutes
	routes := server.GetRoutes()
	assert.NotEmpty(t, routes)

	// 测试 GetRoutesByMethod
	getRoutes := server.GetRoutesByMethod(MethodGet)
	assert.NotEmpty(t, getRoutes)

	// 测试 GetRoutesByPath
	pathRoutes := server.GetRoutesByPath("/items")
	assert.NotEmpty(t, pathRoutes)

	// 测试 HasRoute
	assert.True(t, server.HasRoute(MethodGet, "/items"))
	assert.False(t, server.HasRoute(MethodPost, "/items"))
}

// TestCamelToPath 测试驼峰转路径
func TestCamelToPath(t *testing.T) {
	gen := NewRouterGenerator()

	tests := []struct {
		input    string
		expected string
	}{
		{"UserList", "/user/list"},
		{"GetByID", "/get/by/id"},
		{"", "/"},
		{"ID", "/id"},
		{"UserID", "/user/id"},
		{"GetUserList", "/get/user/list"},
	}

	for _, tt := range tests {
		result := gen.camelToPath(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

// TestAdapterRegistry 测试适配器注册表
// Deprecated: 适配器已移至独立子包，此测试已过时
func TestAdapterRegistry(t *testing.T) {
	t.Skip("Adapter registry is deprecated, adapters are now in separate subpackages")
}

// TestError 测试错误类型
func TestError(t *testing.T) {
	err := NewError(http.StatusNotFound, "not found")
	assert.Equal(t, http.StatusNotFound, err.Code)
	assert.Equal(t, "not found", err.Message)
	assert.Equal(t, "not found", err.Error())

	// 测试带包装错误
	wrappedErr := NewError(http.StatusInternalServerError, "server error", assert.AnError)
	assert.Equal(t, "server error: "+assert.AnError.Error(), wrappedErr.Error())
	assert.Equal(t, assert.AnError, wrappedErr.Unwrap())

	// 测试 ToOption
	opt := wrappedErr.ToOption()
	assert.True(t, opt.IsPresent())

	// 测试 nil error 的 ToOption
	var nilErr *Error = nil
	assert.True(t, nilErr.ToOption().IsAbsent())
}

// TestErrorHelpers 测试错误辅助函数
func TestErrorHelpers(t *testing.T) {
	assert.True(t, IsAdapterNotFound(ErrAdapterNotFound))
	assert.False(t, IsAdapterNotFound(ErrInvalidEndpoint))

	assert.True(t, IsInvalidEndpoint(ErrInvalidEndpoint))
	assert.False(t, IsInvalidEndpoint(ErrAdapterNotFound))
}
