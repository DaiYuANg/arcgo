// Package main demonstrates httpx authentication and documented security schemes.
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
	"github.com/DaiYuANg/arcgo/pkg/randomport"
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

	server := newAuthServer()
	registerAuthRoutes(server)

	port := randomport.MustFind()
	addr := fmt.Sprintf(":%d", port)
	logger.Info("example server starting",
		slog.String("example", "auth"),
		slog.String("address", addr),
		slog.String("openapi", fmt.Sprintf("http://localhost%s/openapi.json", addr)),
		slog.String("docs", fmt.Sprintf("http://localhost%s/docs", addr)),
		slog.String("curl", fmt.Sprintf("curl http://localhost%s/api/secure/profile -H \"Authorization: Bearer demo\" -H \"X-Request-Id: req-1\"", addr)),
	)

	if err := server.ListenPort(port); err != nil {
		logger.Error("server exited with error", slog.String("error", err.Error()))
		closeLogger()
		os.Exit(1)
	}
	closeLogger()
}

func newAuthServer() httpx.ServerRuntime {
	stdAdapter := std.New(nil, adapter.HumaOptions{
		DocsPath:     "/docs",
		OpenAPIPath:  "/openapi.json",
		DocsRenderer: httpx.DocsRendererScalar,
	})

	return httpx.New(
		httpx.WithAdapter(stdAdapter),
		httpx.WithBasePath("/api"),
		httpx.WithOpenAPIInfo("httpx auth example", "1.0.0", "Authentication, security schemes, and custom headers"),
		httpx.WithSecurity(httpx.SecurityOptions{
			Schemes: httpx.SecuritySchemes(map[string]*huma.SecurityScheme{
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
			}),
		}),
	)
}

func registerAuthRoutes(server httpx.ServerRuntime) {
	server.RegisterGlobalHeader(&huma.Param{
		Name:        "X-Request-Id",
		In:          "header",
		Description: "request correlation id",
		Schema:      &huma.Schema{Type: "string"},
	})

	httpx.MustGet(server, "/health", func(_ context.Context, _ *struct{}) (*healthOutput, error) {
		out := &healthOutput{}
		out.Body.Status = "ok"
		return out, nil
	}, huma.OperationTags("system"))

	secure := server.Group("/secure")
	secure.RegisterTags(httpx.TagDefinitions(
		&huma.Tag{Name: "auth", Description: "Authentication examples"},
	))
	secure.DefaultTags(httpx.Tags("auth"))
	secure.DefaultSecurity(httpx.SecurityRequirements(
		httpx.SecurityRequirement("BearerAuth"),
		httpx.SecurityRequirement("ApiKeyAuth"),
	))
	secure.DefaultDescription("Endpoints demonstrating documented authentication headers")

	httpx.MustGroupGet(secure, "/profile", func(_ context.Context, input *profileInput) (*profileOutput, error) {
		return buildProfileOutput(input), nil
	}, func(op *huma.Operation) {
		op.Summary = "Get current profile"
	})
}

func buildProfileOutput(input *profileInput) *profileOutput {
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

	return out
}
