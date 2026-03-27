package gin_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DaiYuANg/arcgo/httpx/adapter"
	ginadapter "github.com/DaiYuANg/arcgo/httpx/adapter/gin"
	ginframework "github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestNew_UsesProvidedEngine(t *testing.T) {
	ginframework.SetMode(ginframework.TestMode)
	engine := ginframework.New()
	a := ginadapter.New(engine)

	assert.Same(t, engine, a.Router())
}

func TestNew_AppliesDocsPaths(t *testing.T) {
	ginframework.SetMode(ginframework.TestMode)
	a := ginadapter.New(nil, adapter.HumaOptions{
		DocsPath:    "/reference",
		OpenAPIPath: "/spec",
	})

	docsReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/reference", http.NoBody)
	docsRec := httptest.NewRecorder()
	a.Router().ServeHTTP(docsRec, docsReq)
	assert.Equal(t, http.StatusOK, docsRec.Code)

	oldDocsReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/docs", http.NoBody)
	oldDocsRec := httptest.NewRecorder()
	a.Router().ServeHTTP(oldDocsRec, oldDocsReq)
	assert.Equal(t, http.StatusNotFound, oldDocsRec.Code)

	specReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/spec.json", http.NoBody)
	specRec := httptest.NewRecorder()
	a.Router().ServeHTTP(specRec, specReq)
	assert.Equal(t, http.StatusOK, specRec.Code)
}

func TestNew_DisablesDocsRoutes(t *testing.T) {
	ginframework.SetMode(ginframework.TestMode)
	a := ginadapter.New(nil, adapter.HumaOptions{DisableDocsRoutes: true})

	docsReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/docs", http.NoBody)
	docsRec := httptest.NewRecorder()
	a.Router().ServeHTTP(docsRec, docsReq)
	assert.Equal(t, http.StatusNotFound, docsRec.Code)

	specReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/openapi.json", http.NoBody)
	specRec := httptest.NewRecorder()
	a.Router().ServeHTTP(specRec, specReq)
	assert.Equal(t, http.StatusNotFound, specRec.Code)
}
