package middleware_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DaiYuANg/arcgo/httpx"
	"github.com/DaiYuANg/arcgo/httpx/middleware"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	tracetest "go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestPrometheusMiddleware_WithHTTPXRoutePattern_UsesRouteTemplateLabel(t *testing.T) {
	server := httpx.New()

	type input struct {
		ID int `path:"id"`
	}
	type output struct {
		Body struct {
			OK bool `json:"ok"`
		} `json:"body"`
	}

	if err := httpx.Get(server, "/users/{id}", func(_ context.Context, in *input) (*output, error) {
		out := &output{}
		out.Body.OK = true
		return out, nil
	}); err != nil {
		t.Fatalf("register route: %v", err)
	}

	handler := middleware.PrometheusMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), middleware.WithHTTPXRoutePattern(server))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/users/42", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status code: %d", rec.Code)
	}

	metricsRec := httptest.NewRecorder()
	metricsReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", http.NoBody)
	middleware.MetricsHandler().ServeHTTP(metricsRec, metricsReq)
	if metricsRec.Code != http.StatusOK {
		t.Fatalf("unexpected metrics status code: %d", metricsRec.Code)
	}

	metricsBody, err := io.ReadAll(metricsRec.Body)
	if err != nil {
		t.Fatalf("read metrics body: %v", err)
	}
	outputText := string(metricsBody)
	if !strings.Contains(outputText, "path=\"/users/{id}\"") {
		t.Fatalf("expected route template label in metrics output, got:\n%s", outputText)
	}
}

func TestOpenTelemetryMiddleware_WithHTTPXRoutePattern_UsesRouteTemplateSpanName(t *testing.T) {
	server := httpx.New()

	type input struct {
		ID int `path:"id"`
	}
	type output struct {
		Body struct {
			OK bool `json:"ok"`
		} `json:"body"`
	}

	if err := httpx.Get(server, "/orders/{id}", func(_ context.Context, in *input) (*output, error) {
		out := &output{}
		out.Body.OK = true
		return out, nil
	}); err != nil {
		t.Fatalf("register route: %v", err)
	}

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown tracer provider: %v", err)
		}
	})

	restore := swapTracerProvider(provider)
	defer restore()

	handler := middleware.OpenTelemetryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}), middleware.WithHTTPXRoutePattern(server))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/orders/99", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status code: %d", rec.Code)
	}

	spans := recorder.Ended()
	if len(spans) == 0 {
		t.Fatal("expected recorded spans")
	}
	span := spans[len(spans)-1]
	if got := span.Name(); got != "HTTP GET /orders/{id}" {
		t.Fatalf("unexpected span name: %q", got)
	}

	var routeAttr string
	for _, attr := range span.Attributes() {
		if string(attr.Key) == "http.route" {
			routeAttr = attr.Value.AsString()
			break
		}
	}
	if routeAttr != "/orders/{id}" {
		t.Fatalf("unexpected http.route attribute: %q", routeAttr)
	}
}

func swapTracerProvider(provider *sdktrace.TracerProvider) func() {
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	return func() {
		otel.SetTracerProvider(previous)
	}
}
