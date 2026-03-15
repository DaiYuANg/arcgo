package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/DaiYuANg/archgo/httpx"
	"github.com/DaiYuANg/archgo/httpx/adapter/std"
	"github.com/DaiYuANg/archgo/httpx/examples/shared"
	"github.com/DaiYuANg/archgo/pkg/randomport"
	"github.com/danielgtaylor/huma/v2"
)

type profileInput struct {
	Authorization string `header:"Authorization"`
	XAPIKey       string `header:"X-API-Key"`
	XRequestID    string `header:"X-Request-Id"`
}

type profileOutput struct {
	Body struct {
		Authorized bool   `json:"authorized"`
		RequestID  string `json:"request_id"`
		AuthMode   string `json:"auth_mode"`
	} `json:"body"`
}

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

	server := httpx.New(
		httpx.WithAdapter(std.New()),
		httpx.WithBasePath("/api"),
		httpx.WithOpenAPIInfo("httpx auth example", "1.0.0", "Authentication, security schemes, and custom headers"),
		httpx.WithDocs(httpx.DocsOptions{
			Enabled:     true,
			DocsPath:    "/docs",
			OpenAPIPath: "/openapi.json",
			Renderer:    httpx.DocsRendererScalar,
		}),
		httpx.WithSecurity(httpx.SecurityOptions{
			Schemes: map[string]*huma.SecurityScheme{
				"BearerAuth": {
					Type:         "http",
					Scheme:       "bearer",
					BearerFormat: "JWT",
					Description:  "Bearer token authentication",
				},
				"ApiKeyAuth": {
					Type:        "apiKey",
					In:          "header",
					Name:        "X-API-Key",
					Description: "API key authentication",
				},
			},
		}),
	)

	server.RegisterGlobalHeader(&huma.Param{
		Name:        "X-Request-Id",
		In:          "header",
		Description: "request correlation id",
		Schema:      &huma.Schema{Type: "string"},
	})

	httpx.MustGet(server, "/health", func(ctx context.Context, input *struct{}) (*healthOutput, error) {
		out := &healthOutput{}
		out.Body.Status = "ok"
		return out, nil
	}, huma.OperationTags("system"))

	secure := server.Group("/secure")
	secure.RegisterTags(
		&huma.Tag{Name: "auth", Description: "Authentication examples"},
	)
	secure.DefaultTags("auth")
	secure.DefaultSecurity(
		map[string][]string{"BearerAuth": {}},
		map[string][]string{"ApiKeyAuth": {}},
	)
	secure.DefaultDescription("Endpoints demonstrating documented authentication headers")

	httpx.MustGroupGet(secure, "/profile", func(ctx context.Context, input *profileInput) (*profileOutput, error) {
		out := &profileOutput{}
		out.Body.RequestID = input.XRequestID

		switch {
		case input.Authorization != "":
			out.Body.Authorized = true
			out.Body.AuthMode = "bearer"
		case input.XAPIKey != "":
			out.Body.Authorized = true
			out.Body.AuthMode = "api-key"
		default:
			out.Body.Authorized = false
			out.Body.AuthMode = "anonymous"
		}

		return out, nil
	}, func(op *huma.Operation) {
		op.Summary = "Get current profile"
	})

	port := randomport.MustFind()
	addr := fmt.Sprintf(":%d", port)
	logger.Info("example server starting",
		slog.String("example", "auth"),
		slog.String("address", addr),
		slog.String("openapi", fmt.Sprintf("http://localhost%s/openapi.json", addr)),
		slog.String("docs", fmt.Sprintf("http://localhost%s/docs", addr)),
		slog.String("curl", fmt.Sprintf("curl http://localhost%s/api/secure/profile -H \"Authorization: Bearer demo\" -H \"X-Request-Id: req-1\"", addr)),
	)

	if err := server.ListenAndServe(addr); err != nil {
		logger.Error("server exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
