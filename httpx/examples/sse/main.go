package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/DaiYuANg/arcgo/httpx"
	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/DaiYuANg/arcgo/httpx/adapter/std"
	"github.com/DaiYuANg/arcgo/httpx/examples/shared"
	"github.com/DaiYuANg/arcgo/pkg/randomport"
	"github.com/danielgtaylor/huma/v2"
)

type streamInput struct {
	Count int `query:"count"`
}

type tickEvent struct {
	Index int    `json:"index"`
	At    string `json:"at"`
}

type doneEvent struct {
	Message string `json:"message"`
}

func main() {
	logger, closeLogger, err := shared.NewLogger()
	if err != nil {
		panic(err)
	}
	defer closeLogger()

	a := std.NewWithOptions(std.Options{
		Huma: adapter.HumaOptions{
			Title:       "httpx SSE example",
			Version:     "1.0.0",
			Description: "SSE streaming with typed events",
			DocsPath:    "/docs",
			OpenAPIPath: "/openapi.json",
		},
	})

	s := httpx.New(httpx.WithAdapter(a))

	httpx.MustRouteSSEWithPolicies(s, httpx.MethodGet, "/events", map[string]any{
		"tick": tickEvent{},
		"done": doneEvent{},
	}, func(ctx context.Context, input *streamInput, send httpx.SSESender) {
		count := input.Count
		if count <= 0 {
			count = 3
		}

		for i := 1; i <= count; i++ {
			select {
			case <-ctx.Done():
				return
			default:
			}
			if err := send(httpx.SSEMessage{
				ID:   i,
				Data: tickEvent{Index: i, At: time.Now().UTC().Format(time.RFC3339Nano)},
			}); err != nil {
				return
			}
		}

		_ = send(httpx.SSEMessage{
			ID:   count + 1,
			Data: doneEvent{Message: "stream completed"},
		})
	}, httpx.SSEPolicyOperation[streamInput](huma.OperationTags("streaming")))

	port := randomport.MustFind()
	addr := fmt.Sprintf(":%d", port)
	logger.Info("example server starting",
		slog.String("example", "sse"),
		slog.String("address", addr),
		slog.String("openapi", fmt.Sprintf("http://localhost%s/openapi.json", addr)),
		slog.String("docs", fmt.Sprintf("http://localhost%s/docs", addr)),
		slog.String("stream_url", fmt.Sprintf("http://localhost%s/events?count=5", addr)),
		slog.String("curl", fmt.Sprintf("curl -N http://localhost%s/events?count=5", addr)),
	)

	if err := s.ListenAndServe(addr); err != nil {
		logger.Error("server exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
