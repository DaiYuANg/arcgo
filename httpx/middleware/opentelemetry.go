// OpenTelemetry 追踪中间件
package middleware

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("github.com/DaiYuANg/arcgo/httpx")

// OpenTelemetryMiddleware OpenTelemetry 追踪中间件
func OpenTelemetryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 从请求中提取 trace context
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		// 创建 span
		opts := []trace.SpanStartOption{
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPMethod(r.Method),
				semconv.HTTPTarget(r.URL.EscapedPath()),
				semconv.HTTPURL(r.URL.String()),
				semconv.NetHostName(r.Host),
			),
		}

		ctx, span := tracer.Start(ctx, "HTTP "+r.Method+" "+r.URL.Path, opts...)
		defer span.End()

		// 包装 ResponseWriter 以捕获状态码
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// 将 trace context 注入到请求中
		r = r.WithContext(ctx)

		// 处理请求
		next.ServeHTTP(wrapped, r)

		// 记录响应状态码
		span.SetAttributes(semconv.HTTPStatusCode(wrapped.statusCode))

		// 记录延迟
		span.SetAttributes(
			attribute.Int64("http.response_time_ms", time.Since(start).Milliseconds()),
		)
	})
}

// InjectTraceContext 将 trace context 注入到 HTTP 请求头
func InjectTraceContext(ctx context.Context, header http.Header) {
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(header))
}

// ExtractTraceContext 从 HTTP 请求头提取 trace context
func ExtractTraceContext(ctx context.Context, header http.Header) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(header))
}
