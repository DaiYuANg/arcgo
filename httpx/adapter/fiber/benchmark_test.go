package fiber

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func benchmarkAdapterWithRoute(b *testing.B) *Adapter {
	b.Helper()

	a := New(nil)
	a.Handle(http.MethodGet, "/ping", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		_ = ctx
		_ = r
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
		return nil
	})
	return a
}

func BenchmarkAdapterTestRequest(b *testing.B) {
	a := benchmarkAdapterWithRoute(b)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		resp, err := a.Router().Test(req, -1)
		if err != nil {
			b.Fatalf("fiber test request failed: %v", err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b.Fatalf("unexpected status code: %d", resp.StatusCode)
		}
	}
}
