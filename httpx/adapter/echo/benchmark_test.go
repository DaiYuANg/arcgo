package echo

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

func BenchmarkAdapterServeHTTP(b *testing.B) {
	a := benchmarkAdapterWithRoute(b)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		w := httptest.NewRecorder()
		a.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("unexpected status code: %d", w.Code)
		}
	}
}
