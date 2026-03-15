// OpenTelemetry documents related behavior.
package middleware

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("github.com/DaiYuANg/archgo/httpx")

// OpenTelemetryMiddleware documents related behavior.
func OpenTelemetryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Note.
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		// Note.
		opts := []trace.SpanStartOption{
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.request.method", r.Method),
				attribute.String("url.path", r.URL.EscapedPath()),
				attribute.String("url.full", r.URL.String()),
				attribute.String("server.address", r.Host),
			),
		}

		ctx, span := tracer.Start(ctx, "HTTP "+r.Method+" "+r.URL.Path, opts...)
		defer span.End()

		// Note.
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Note.
		r = r.WithContext(ctx)

		// Note.
		next.ServeHTTP(wrapped, r)

		// Note.
		span.SetAttributes(attribute.Int("http.response.status_code", wrapped.statusCode))

		// Note.
		span.SetAttributes(
			attribute.Int64("http.response_time_ms", time.Since(start).Milliseconds()),
		)
	})
}

// InjectTraceContext documents related behavior.
func InjectTraceContext(ctx context.Context, header http.Header) {
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(header))
}

// ExtractTraceContext documents related behavior.
func ExtractTraceContext(ctx context.Context, header http.Header) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(header))
}
