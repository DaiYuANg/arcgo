package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/DaiYuANg/arcgo/httpx/adapter/std"
	"github.com/stretchr/testify/assert"
)

func TestServer_Docs1(t *testing.T) {
	// 模拟 auth example 的场景：只使用 WithOpenAPIInfo
	a := std.New(adapter.HumaOptions{
		Title:       "test api",
		Version:     "1.0.0",
		Description: "test",
	})

	s := newServer(
		WithAdapter(a),
		WithOpenAPIInfo("My API", "1.0.0", "My Description"),
	)

	t.Logf("s.humaOptions: %+v", s.humaOptions)
	t.Logf("isZeroDocsOptions: %v", isZeroDocsOptions(s.humaOptions))

	docsReq := httptest.NewRequest(http.MethodGet, "/docs", nil)
	docsRec := httptest.NewRecorder()
	s.ServeHTTP(docsRec, docsReq)
	t.Logf("/docs status: %d", docsRec.Code)
	assert.Equal(t, http.StatusOK, docsRec.Code)
}

func TestServer_Docs2(t *testing.T) {
	// 最简单的场景：只用 std.New()
	a := std.New()

	s := newServer(
		WithAdapter(a),
	)

	t.Logf("s.humaOptions: %+v", s.humaOptions)
	t.Logf("isZeroDocsOptions: %v", isZeroDocsOptions(s.humaOptions))

	docsReq := httptest.NewRequest(http.MethodGet, "/docs", nil)
	docsRec := httptest.NewRecorder()
	s.ServeHTTP(docsRec, docsReq)
	t.Logf("/docs status: %d", docsRec.Code)
	assert.Equal(t, http.StatusOK, docsRec.Code)
}
