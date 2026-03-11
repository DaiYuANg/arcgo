package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

type benchmarkPingOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

func benchmarkServerWithPingRoute(b *testing.B) *Server {
	b.Helper()

	server := newServer()
	err := Get(server, "/ping", func(ctx context.Context, input *struct{}) (*benchmarkPingOutput, error) {
		_ = ctx
		_ = input
		out := &benchmarkPingOutput{}
		out.Body.Message = "pong"
		return out, nil
	})
	if err != nil {
		b.Fatal(err)
	}
	return server
}

func BenchmarkServerRegisterGet(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		server := newServer()
		path := "/bench/" + strconv.Itoa(i)
		err := Get(server, path, func(ctx context.Context, input *struct{}) (*benchmarkPingOutput, error) {
			_ = ctx
			_ = input
			out := &benchmarkPingOutput{}
			out.Body.Message = "ok"
			return out, nil
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkServerServeHTTPGet(b *testing.B) {
	server := benchmarkServerWithPingRoute(b)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("unexpected status code: %d", w.Code)
		}
	}
}

func BenchmarkPathNormalizeAndJoin(b *testing.B) {
	base := " /api/v1/ "
	path := "users/{id}"

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		prefix := normalizeRoutePrefix(base)
		fullPath := joinRoutePath(prefix, path)
		if fullPath == "" {
			b.Fatal("unexpected empty path")
		}
	}
}
