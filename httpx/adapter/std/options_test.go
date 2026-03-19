package std

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestNew_UsesProvidedRouter(t *testing.T) {
	router := chi.NewMux()
	a := New(router)

	assert.Same(t, router, a.Router())
}

func TestNew_UsesPreconfiguredRouterMiddleware(t *testing.T) {
	router := chi.NewMux()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test-Middleware", "ok")
			next.ServeHTTP(w, r)
		})
	})

	a := New(router)

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()
	a.Router().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Header().Get("X-Test-Middleware"))
}

func TestNew_AppliesDocsPaths(t *testing.T) {
	a := New(nil, adapter.HumaOptions{
		DocsPath:    "/reference",
		OpenAPIPath: "/spec",
	})

	docsReq := httptest.NewRequest(http.MethodGet, "/reference", nil)
	docsRec := httptest.NewRecorder()
	a.Router().ServeHTTP(docsRec, docsReq)
	assert.Equal(t, http.StatusOK, docsRec.Code)

	oldDocsReq := httptest.NewRequest(http.MethodGet, "/docs", nil)
	oldDocsRec := httptest.NewRecorder()
	a.Router().ServeHTTP(oldDocsRec, oldDocsReq)
	assert.Equal(t, http.StatusNotFound, oldDocsRec.Code)

	specReq := httptest.NewRequest(http.MethodGet, "/spec.json", nil)
	specRec := httptest.NewRecorder()
	a.Router().ServeHTTP(specRec, specReq)
	assert.Equal(t, http.StatusOK, specRec.Code)
}

func TestNew_DisablesDocsRoutes(t *testing.T) {
	a := New(nil, adapter.HumaOptions{DisableDocsRoutes: true})

	docsReq := httptest.NewRequest(http.MethodGet, "/docs", nil)
	docsRec := httptest.NewRecorder()
	a.Router().ServeHTTP(docsRec, docsReq)
	assert.Equal(t, http.StatusNotFound, docsRec.Code)

	specReq := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	specRec := httptest.NewRecorder()
	a.Router().ServeHTTP(specRec, specReq)
	assert.Equal(t, http.StatusNotFound, specRec.Code)
}
