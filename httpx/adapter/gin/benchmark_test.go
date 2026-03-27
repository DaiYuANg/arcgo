//go:build !no_gin

package gin_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	ginadapter "github.com/DaiYuANg/arcgo/httpx/adapter/gin"
	"github.com/danielgtaylor/huma/v2"
	ginframework "github.com/gin-gonic/gin"
)

type benchmarkOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

func benchmarkAdapterWithRoute(b *testing.B) *ginadapter.Adapter {
	b.Helper()

	ginframework.SetMode(ginframework.TestMode)
	a := ginadapter.New(nil)
	huma.Register(a.HumaAPI(), huma.Operation{
		OperationID: "ping",
		Method:      http.MethodGet,
		Path:        "/ping",
	}, func(_ context.Context, _ *struct{}) (*benchmarkOutput, error) {
		out := &benchmarkOutput{}
		out.Body.Message = "pong"
		return out, nil
	})
	return a
}

func BenchmarkAdapterRouterServeHTTP(b *testing.B) {
	a := benchmarkAdapterWithRoute(b)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ping", http.NoBody)
		w := httptest.NewRecorder()
		a.Router().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("unexpected status code: %d", w.Code)
		}
	}
}
