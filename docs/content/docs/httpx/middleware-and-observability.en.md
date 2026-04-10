---
title: 'httpx Middleware and Observability'
linkTitle: 'middleware-and-observability'
description: 'Prometheus and OpenTelemetry middleware for httpx'
weight: 6
---

## Middleware and observability

`httpx/middleware` provides small reusable pieces around standard `net/http` handlers.

The main observability helpers today are:

- `PrometheusMiddleware`
- `OpenTelemetryMiddleware`

## Route-aware labels and span names

When metrics or traces use the raw request path, dynamic IDs can create high-cardinality labels:

- `/users/1`
- `/users/2`
- `/users/3`

For `httpx` services you usually want the route template instead:

- `/users/{id}`

`httpx/middleware` supports this through `WithHTTPXRoutePattern(server)`.

## Prometheus example

```go
server := httpx.New(...)

handler := middleware.PrometheusMiddleware(
	serverRouter,
	middleware.WithHTTPXRoutePattern(server),
)
```

With route pattern resolution enabled, metrics use the registered route template instead of the raw path.

## OpenTelemetry example

```go
handler := middleware.OpenTelemetryMiddleware(
	serverRouter,
	middleware.WithHTTPXRoutePattern(server),
)
```

With route pattern resolution enabled:

- span names use the route template
- `http.route` is attached to the span

## Why this matters

- lower metric label cardinality
- cleaner dashboards
- more stable tracing aggregation
- better parity with server-side route registration

## Notes

- The middleware package is optional.
- It wraps standard handlers, not typed `httpx` route registration directly.
- Route pattern resolution is opt-in. Existing middleware behavior stays unchanged unless you pass the option.
