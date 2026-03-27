package fiber_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	fiberadapter "github.com/DaiYuANg/arcgo/httpx/adapter/fiber"
	"github.com/danielgtaylor/huma/v2"
)

type benchmarkOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

func benchmarkAdapterWithRoute(b *testing.B) *fiberadapter.Adapter {
	b.Helper()

	a := fiberadapter.New(nil)
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

func BenchmarkAdapterTestRequest(b *testing.B) {
	a := benchmarkAdapterWithRoute(b)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ping", http.NoBody)
		resp, err := a.Router().Test(req, -1)
		if err != nil {
			b.Fatalf("fiber test request failed: %v", err)
		}
		if _, err := io.Copy(io.Discard, resp.Body); err != nil {
			b.Fatalf("discard response body failed: %v", err)
		}
		if err := resp.Body.Close(); err != nil {
			b.Fatalf("close response body failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			b.Fatalf("unexpected status code: %d", resp.StatusCode)
		}
	}
}
