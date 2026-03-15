//go:build !no_fiber

package fiber_test

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authhttp "github.com/DaiYuANg/archgo/authx/http"
	authfiber "github.com/DaiYuANg/archgo/authx/http/fiber"
	"github.com/DaiYuANg/archgo/authx/http/internal/benchmarksupport"
	"github.com/gofiber/fiber/v2"
)

func BenchmarkRequireCheckCan10kUsers10kPermissions(b *testing.B) {
	runInProcessBench(b, 5051, authfiber.Require)
}

func BenchmarkRequireFastCheckCan10kUsers10kPermissions(b *testing.B) {
	runInProcessBench(b, 5053, authfiber.RequireFast)
}

func BenchmarkRequireCheckCan10kUsers10kPermissionsRealHTTP(b *testing.B) {
	runRealHTTPBench(b, 5052, authfiber.Require)
}

func BenchmarkRequireFastCheckCan10kUsers10kPermissionsRealHTTP(b *testing.B) {
	runRealHTTPBench(b, 5054, authfiber.RequireFast)
}

func runInProcessBench(
	b *testing.B,
	seed uint64,
	builder func(*authhttp.Guard, ...authfiber.Option) fiber.Handler,
) {
	dataset := benchmarksupport.NewDataset(seed, 10_000, 10_000, 16, 2_048)
	guard := benchmarksupport.NewGuard(dataset)

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(builder(guard))
	app.Get("/authz/benchmark", func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusNoContent)
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		query := dataset.Queries[i%len(dataset.Queries)]
		req := httptest.NewRequest(http.MethodGet, "/authz/benchmark", nil)
		req.Header.Set(benchmarksupport.HeaderUserID, query.UserID)
		req.Header.Set(benchmarksupport.HeaderAction, query.Action)
		req.Header.Set(benchmarksupport.HeaderResource, query.Resource)

		resp, err := app.Test(req, -1)
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

func runRealHTTPBench(
	b *testing.B,
	seed uint64,
	builder func(*authhttp.Guard, ...authfiber.Option) fiber.Handler,
) {
	dataset := benchmarksupport.NewDataset(seed, 10_000, 10_000, 16, 2_048)
	guard := benchmarksupport.NewGuard(dataset)

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(builder(guard))
	app.Get("/authz/benchmark", func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusNoContent)
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("listen failed: %v", err)
	}
	defer func() {
		_ = listener.Close()
	}()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- app.Listener(listener)
	}()

	b.Cleanup(func() {
		_ = app.Shutdown()
		select {
		case err := <-serverErr:
			if err != nil {
				b.Fatalf("fiber server failed: %v", err)
			}
		case <-time.After(500 * time.Millisecond):
		}
	})

	baseURL := "http://" + listener.Addr().String()
	waitForFiberReady(b, baseURL)

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		query := dataset.Queries[i%len(dataset.Queries)]
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/authz/benchmark", baseURL), nil)
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

func waitForFiberReady(b *testing.B, baseURL string) {
	b.Helper()

	client := &http.Client{
		Timeout: 300 * time.Millisecond,
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		req, err := http.NewRequest(http.MethodGet, baseURL+"/authz/benchmark", nil)
		if err != nil {
			b.Fatalf("create readiness request failed: %v", err)
		}
		req.Header.Set(benchmarksupport.HeaderUserID, "warmup")
		req.Header.Set(benchmarksupport.HeaderAction, "warmup")
		req.Header.Set(benchmarksupport.HeaderResource, "warmup")

		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			return
		}

		time.Sleep(25 * time.Millisecond)
	}

	b.Fatal("fiber server readiness timeout")
}
