package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestServer_BugRepro_DocsEnabled 复现用户报告的 bug：
// 配置 WithDocs 后 /docs 和 /openapi.json 返回 404
func TestServer_BugRepro_DocsEnabled(t *testing.T) {
	// 精确复现用户的配置
	s := newServer(
		WithDocs(DocsOptions{
			Enabled:     true,
			DocsPath:    "/docs",
			OpenAPIPath: "/openapi.json",
			Renderer:    DocsRendererScalar,
		}),
	)

	// 测试 /docs
	docsReq := httptest.NewRequest(http.MethodGet, "/docs", nil)
	docsRec := httptest.NewRecorder()
	s.ServeHTTP(docsRec, docsReq)
	t.Logf("/docs status: %d, body len: %d", docsRec.Code, docsRec.Body.Len())
	assert.Equal(t, http.StatusOK, docsRec.Code, "/docs should return 200")

	// 测试 /openapi.json
	openAPIReq := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	openAPIRec := httptest.NewRecorder()
	s.ServeHTTP(openAPIRec, openAPIReq)
	t.Logf("/openapi.json status: %d, body len: %d", openAPIRec.Code, openAPIRec.Body.Len())
	assert.Equal(t, http.StatusOK, openAPIRec.Code, "/openapi.json should return 200")
}

// TestServer_BugRepro_DocsWithoutConfig 验证不配置 WithDocs 时正常工作
func TestServer_BugRepro_DocsWithoutConfig(t *testing.T) {
	// 不配置 WithDocs
	s := newServer()

	// 测试 /docs
	docsReq := httptest.NewRequest(http.MethodGet, "/docs", nil)
	docsRec := httptest.NewRecorder()
	s.ServeHTTP(docsRec, docsReq)
	t.Logf("/docs status: %d, body len: %d", docsRec.Code, docsRec.Body.Len())
	assert.Equal(t, http.StatusOK, docsRec.Code, "/docs should return 200")

	// 测试 /openapi.json
	openAPIReq := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	openAPIRec := httptest.NewRecorder()
	s.ServeHTTP(openAPIRec, openAPIReq)
	t.Logf("/openapi.json status: %d, body len: %d", openAPIRec.Code, openAPIRec.Body.Len())
	assert.Equal(t, http.StatusOK, openAPIRec.Code, "/openapi.json should return 200")
}

// TestServer_BugRepro_DocsWithOpenAPIInfo 测试 WithOpenAPIInfo + WithDocs 的组合
func TestServer_BugRepro_DocsWithOpenAPIInfo(t *testing.T) {
	// 测试 WithOpenAPIInfo + WithDocs 的组合
	s := newServer(
		WithOpenAPIInfo("My API", "1.0.0", "My Description"),
		WithDocs(DocsOptions{
			Enabled:     true,
			DocsPath:    "/docs",
			OpenAPIPath: "/openapi.json",
			Renderer:    DocsRendererScalar,
		}),
	)

	// 测试 /docs
	docsReq := httptest.NewRequest(http.MethodGet, "/docs", nil)
	docsRec := httptest.NewRecorder()
	s.ServeHTTP(docsRec, docsReq)
	t.Logf("/docs status: %d, body len: %d", docsRec.Code, docsRec.Body.Len())
	assert.Equal(t, http.StatusOK, docsRec.Code, "/docs should return 200")

	// 测试 /openapi.json
	openAPIReq := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	openAPIRec := httptest.NewRecorder()
	s.ServeHTTP(openAPIRec, openAPIReq)
	t.Logf("/openapi.json status: %d, body len: %d", openAPIRec.Code, openAPIRec.Body.Len())
	assert.Equal(t, http.StatusOK, openAPIRec.Code, "/openapi.json should return 200")
}

// TestServer_BugRepro_OnlyWithOpenAPIInfo 测试只用 WithOpenAPIInfo（不用 WithDocs）
func TestServer_BugRepro_OnlyWithOpenAPIInfo(t *testing.T) {
	// 只用 WithOpenAPIInfo，不用 WithDocs
	s := newServer(
		WithOpenAPIInfo("My API", "1.0.0", "My Description"),
	)

	// 测试 /docs
	docsReq := httptest.NewRequest(http.MethodGet, "/docs", nil)
	docsRec := httptest.NewRecorder()
	s.ServeHTTP(docsRec, docsReq)
	t.Logf("/docs status: %d, body len: %d", docsRec.Code, docsRec.Body.Len())
	assert.Equal(t, http.StatusOK, docsRec.Code, "/docs should return 200")

	// 测试 /openapi.json
	openAPIReq := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	openAPIRec := httptest.NewRecorder()
	s.ServeHTTP(openAPIRec, openAPIReq)
	t.Logf("/openapi.json status: %d, body len: %d", openAPIRec.Code, openAPIRec.Body.Len())
	assert.Equal(t, http.StatusOK, openAPIRec.Code, "/openapi.json should return 200")
}

// TestServer_BugRepro_DocsDisabled 测试 Enabled: false 的情况
func TestServer_BugRepro_DocsDisabled(t *testing.T) {
	s := newServer(
		WithDocs(DocsOptions{
			Enabled:     false,
			DocsPath:    "/docs",
			OpenAPIPath: "/openapi.json",
			Renderer:    DocsRendererScalar,
		}),
	)

	// 测试 /docs 应该 404
	docsReq := httptest.NewRequest(http.MethodGet, "/docs", nil)
	docsRec := httptest.NewRecorder()
	s.ServeHTTP(docsRec, docsReq)
	t.Logf("/docs status: %d", docsRec.Code)
	assert.Equal(t, http.StatusNotFound, docsRec.Code, "/docs should return 404 when disabled")
}

// TestServer_BugRepro_DocsEnabledDefaultPaths 测试 Enabled: true 但使用默认路径
func TestServer_BugRepro_DocsEnabledDefaultPaths(t *testing.T) {
	s := newServer(
		WithDocs(DocsOptions{
			Enabled:  true,
			Renderer: DocsRendererScalar,
		}),
	)

	// 测试 /docs
	docsReq := httptest.NewRequest(http.MethodGet, "/docs", nil)
	docsRec := httptest.NewRecorder()
	s.ServeHTTP(docsRec, docsReq)
	assert.Equal(t, http.StatusOK, docsRec.Code, "/docs should return 200")

	// 测试 /openapi
	openAPIReq := httptest.NewRequest(http.MethodGet, "/openapi", nil)
	openAPIRec := httptest.NewRecorder()
	s.ServeHTTP(openAPIRec, openAPIReq)
	assert.Equal(t, http.StatusOK, openAPIRec.Code, "/openapi should return 200")

	// 测试 /openapi.json
	openAPIJSONReq := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	openAPIJSONRec := httptest.NewRecorder()
	s.ServeHTTP(openAPIJSONRec, openAPIJSONReq)
	assert.Equal(t, http.StatusOK, openAPIJSONRec.Code, "/openapi.json should return 200")
}
