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

var tracer = otel.Tracer("github.com/DaiYuANg/arcgo/httpx")

// OpenTelemetryMiddleware records request spans, optionally normalized by route pattern.
func OpenTelemetryMiddleware(next http.Handler, opts ...Option) http.Handler {
	cfg := applyOptions(opts)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		route := routePattern(r, cfg)

		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		ctx, span := tracer.Start(ctx, "HTTP "+r.Method+" "+route,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.request.method", r.Method),
				attribute.String("http.route", route),
				attribute.String("url.path", requestEscapedPath(r)),
				attribute.String("url.full", requestURL(r)),
				attribute.String("server.address", r.Host),
			),
		)
		defer span.End()

		wrapped := newResponseWriter(w)
		r = r.WithContext(ctx)
		next.ServeHTTP(wrapped, r)

		span.SetAttributes(attribute.Int("http.response.status_code", wrapped.statusCode))
		span.SetAttributes(attribute.Int64("http.response_time_ms", time.Since(start).Milliseconds()))
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
