package std_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	stdadapter "github.com/DaiYuANg/arcgo/httpx/adapter/std"
	"github.com/danielgtaylor/huma/v2"
)

func benchmarkAdapterWithRoute(b *testing.B) *stdadapter.Adapter {
	b.Helper()

	a := stdadapter.New(nil)
	huma.Register(a.HumaAPI(), huma.Operation{
		OperationID: "ping",
		Method:      http.MethodGet,
		Path:        "/ping",
	}, func(_ context.Context, _ *struct{}) (*pingOutput, error) {
		out := &pingOutput{}
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
