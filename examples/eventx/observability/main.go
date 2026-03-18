package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/DaiYuANg/arcgo/eventx"
	"github.com/DaiYuANg/arcgo/httpx"
	"github.com/DaiYuANg/arcgo/httpx/adapter/std"
	"github.com/DaiYuANg/arcgo/observabilityx"
	otelobs "github.com/DaiYuANg/arcgo/observabilityx/otel"
	promobs "github.com/DaiYuANg/arcgo/observabilityx/prometheus"
)

type userCreated struct {
	ID int
}

func (e userCreated) Name() string {
	return "user.created"
}

func main() {
	prom := promobs.New(promobs.WithNamespace("eventx_example"))
	obs := observabilityx.Multi(otelobs.New(), prom)

	bus := eventx.New(
		eventx.WithObservability(obs),
		eventx.WithAntsPool(2),
		eventx.WithAsyncQueueSize(16),
		eventx.WithMiddleware(eventx.RecoverMiddleware()),
	)
	defer func() { _ = bus.Close() }()

	unsubscribe, err := eventx.Subscribe(bus, func(ctx context.Context, evt userCreated) error {
		fmt.Println("user created:", evt.ID)
		return nil
	})
	if err != nil {
		panic(err)
	}
	defer unsubscribe()

	if err := bus.Publish(context.Background(), userCreated{ID: 1}); err != nil {
		panic(err)
	}
	if err := bus.PublishAsync(context.Background(), userCreated{ID: 2}); err != nil {
		panic(err)
	}

	metricsServer := httpx.New(
		httpx.WithAdapter(std.New()),
		httpx.WithOpenAPIDocs(false),
	)
	metricsServer.Adapter().Handle(httpx.MethodGet, "/metrics", func(
		ctx context.Context,
		w http.ResponseWriter,
		r *http.Request,
	) error {
		_ = ctx
		prom.Handler().ServeHTTP(w, r)
		return nil
	})

	fmt.Println("httpx metrics route registered: GET /metrics")
	_ = metricsServer
}
