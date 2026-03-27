//go:build !no_fiber

package fiber_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DaiYuANg/arcgo/httpx/adapter"
	fiberadapter "github.com/DaiYuANg/arcgo/httpx/adapter/fiber"
	fiberframework "github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_UsesProvidedApp(t *testing.T) {
	external := fiberframework.New()
	a := fiberadapter.New(external)

	assert.Same(t, external, a.Router())
}

func TestNew_AppliesDocsPaths(t *testing.T) {
	a := fiberadapter.New(nil, adapter.HumaOptions{
		DocsPath:    "/reference",
		OpenAPIPath: "/spec",
	})

	assertTestStatus(
		t,
		a,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/reference", http.NoBody),
		http.StatusOK,
	)
	assertTestStatus(
		t,
		a,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/docs", http.NoBody),
		http.StatusNotFound,
	)
	assertTestStatus(
		t,
		a,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/spec.json", http.NoBody),
		http.StatusOK,
	)
}

func TestNew_DisablesDocsRoutes(t *testing.T) {
	a := fiberadapter.New(nil, adapter.HumaOptions{DisableDocsRoutes: true})

	assertTestStatus(
		t,
		a,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/docs", http.NoBody),
		http.StatusNotFound,
	)
	assertTestStatus(
		t,
		a,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/openapi.json", http.NoBody),
		http.StatusNotFound,
	)
}

func assertTestStatus(t *testing.T, a *fiberadapter.Adapter, req *http.Request, expected int) {
	t.Helper()

	resp, err := a.Router().Test(req, -1)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, resp.Body.Close())
	}()
	assert.Equal(t, expected, resp.StatusCode)
}
