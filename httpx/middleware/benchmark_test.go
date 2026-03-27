package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DaiYuANg/arcgo/httpx/middleware"
)

func BenchmarkPrometheusMiddleware(b *testing.B) {
	handler := middleware.PrometheusMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics-demo", http.NoBody)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			b.Fatalf("unexpected status code: %d", w.Code)
		}
	}
}

func BenchmarkMetricsHandlerServeHTTP(b *testing.B) {
	handler := middleware.MetricsHandler()

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", http.NoBody)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("unexpected status code: %d", w.Code)
		}
	}
}
