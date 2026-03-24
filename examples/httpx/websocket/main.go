package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/DaiYuANg/arcgo/examples/httpx/shared"
	"github.com/DaiYuANg/arcgo/httpx"
	"github.com/DaiYuANg/arcgo/httpx/adapter"
	"github.com/DaiYuANg/arcgo/httpx/adapter/std"
	"github.com/DaiYuANg/arcgo/httpx/websocket"
	"github.com/DaiYuANg/arcgo/pkg/randomport"
	"github.com/go-chi/chi/v5"
)

type healthOutput struct {
	Body struct {
		Status string `json:"status"`
	} `json:"body"`
}

func main() {
	logger, closeLogger, err := shared.NewLogger()
	if err != nil {
		panic(err)
	}
	defer closeLogger()

	router := chi.NewRouter()
	a := std.New(router, adapter.HumaOptions{
		Title:       "httpx websocket example",
		Version:     "1.0.0",
		Description: "Typed HTTP routes + websocket echo endpoint",
		DocsPath:    "/docs",
		OpenAPIPath: "/openapi.json",
	})
	s := httpx.New(httpx.WithAdapter(a))

	httpx.MustGet(s, "/health", func(ctx context.Context, input *struct{}) (*healthOutput, error) {
		out := &healthOutput{}
		out.Body.Status = "ok"
		return out, nil
	})

	router.HandleFunc("/ws/echo", websocket.HandlerFunc(func(ctx context.Context, conn websocket.Conn) error {
		for {
			msg, err := conn.Read(ctx)
			if err != nil {
				return err
			}
			if err := conn.Write(msg); err != nil {
				return err
			}
		}
	}, websocket.WithCompression(true)))

	port := randomport.MustFind()
	addr := fmt.Sprintf(":%d", port)
	logger.Info("example server starting",
		slog.String("example", "websocket"),
		slog.String("address", addr),
		slog.String("openapi", fmt.Sprintf("http://localhost%s/openapi.json", addr)),
		slog.String("docs", fmt.Sprintf("http://localhost%s/docs", addr)),
		slog.String("health", fmt.Sprintf("http://localhost%s/health", addr)),
		slog.String("ws", fmt.Sprintf("ws://localhost%s/ws/echo", addr)),
	)

	if err := s.ListenPort(port); err != nil {
		logger.Error("server exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
