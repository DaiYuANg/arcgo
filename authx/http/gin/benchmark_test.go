//go:build !no_gin

package gin_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	authhttp "github.com/DaiYuANg/archgo/authx/http"
	authgin "github.com/DaiYuANg/archgo/authx/http/gin"
	"github.com/DaiYuANg/archgo/authx/http/internal/benchmarksupport"
	"github.com/gin-gonic/gin"
)

func BenchmarkRequireCheckCan10kUsers10kPermissions(b *testing.B) {
	runInProcessBench(b, 3031, authgin.Require)
}

func BenchmarkRequireFastCheckCan10kUsers10kPermissions(b *testing.B) {
	runInProcessBench(b, 3033, authgin.RequireFast)
}

func BenchmarkRequireCheckCan10kUsers10kPermissionsRealHTTP(b *testing.B) {
	runRealHTTPBench(b, 3032, authgin.Require)
}

func BenchmarkRequireFastCheckCan10kUsers10kPermissionsRealHTTP(b *testing.B) {
	runRealHTTPBench(b, 3034, authgin.RequireFast)
}

func runInProcessBench(
	b *testing.B,
	seed uint64,
	builder func(*authhttp.Guard, ...authgin.Option) gin.HandlerFunc,
) {
	gin.SetMode(gin.ReleaseMode)

	dataset := benchmarksupport.NewDataset(seed, 10_000, 10_000, 16, 2_048)
	guard := benchmarksupport.NewGuard(dataset)

	router := gin.New()
	router.Use(builder(guard))
	router.GET("/authz/benchmark", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		query := dataset.Queries[i%len(dataset.Queries)]
		req := httptest.NewRequest(http.MethodGet, "/authz/benchmark", nil)
		req.Header.Set(benchmarksupport.HeaderUserID, query.UserID)
		req.Header.Set(benchmarksupport.HeaderAction, query.Action)
		req.Header.Set(benchmarksupport.HeaderResource, query.Resource)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		expectedStatus := http.StatusNoContent
		if !query.Allowed {
			expectedStatus = http.StatusForbidden
		}
		if w.Code != expectedStatus {
			b.Fatalf("unexpected status: got=%d expected=%d", w.Code, expectedStatus)
		}
	}
}

func runRealHTTPBench(
	b *testing.B,
	seed uint64,
	builder func(*authhttp.Guard, ...authgin.Option) gin.HandlerFunc,
) {
	gin.SetMode(gin.ReleaseMode)

	dataset := benchmarksupport.NewDataset(seed, 10_000, 10_000, 16, 2_048)
	guard := benchmarksupport.NewGuard(dataset)

	router := gin.New()
	router.Use(builder(guard))
	router.GET("/authz/benchmark", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	server := httptest.NewServer(router)
	b.Cleanup(server.Close)

	client := server.Client()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		query := dataset.Queries[i%len(dataset.Queries)]
		req, err := http.NewRequest(http.MethodGet, server.URL+"/authz/benchmark", nil)
		if err != nil {
			b.Fatalf("create request failed: %v", err)
		}
		req.Header.Set(benchmarksupport.HeaderUserID, query.UserID)
		req.Header.Set(benchmarksupport.HeaderAction, query.Action)
		req.Header.Set(benchmarksupport.HeaderResource, query.Resource)

		resp, err := client.Do(req)
		if err != nil {
			b.Fatalf("request failed: %v", err)
		}
		_ = resp.Body.Close()

		expectedStatus := http.StatusNoContent
		if !query.Allowed {
			expectedStatus = http.StatusForbidden
		}
		if resp.StatusCode != expectedStatus {
			b.Fatalf("unexpected status: got=%d expected=%d", resp.StatusCode, expectedStatus)
		}
	}
}
