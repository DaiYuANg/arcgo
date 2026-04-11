// Package main demonstrates organizing httpx docs, security, and group defaults.
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

type healthOutput struct {
	Body struct {
		Status string `json:"status"`
	} `json:"body"`
}

type tenantInput struct {
	ID string `path:"id"`
}

type tenantOutput struct {
	Body struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"body"`
}

func main() {
	logger, closeLogger, err := shared.NewLogger()
	if err != nil {
		panic(err)
	}
	server := newOrganizationServer()
	registerOrganizationRoutes(server)

	port := randomport.MustFind()
	addr := fmt.Sprintf(":%d", port)
	logger.Info("example server starting",
		slog.String("example", "organization"),
		slog.String("address", addr),
		slog.String("openapi", fmt.Sprintf("http://localhost%s/spec.json", addr)),
		slog.String("docs", fmt.Sprintf("http://localhost%s/reference", addr)),
	)

	if err := server.ListenPort(port); err != nil {
		logger.Error("server exited with error", slog.String("error", err.Error()))
		closeLogger()
		os.Exit(1)
	}
	closeLogger()
}

func newOrganizationServer() httpx.ServerRuntime {
	stdAdapter := std.New(nil, adapter.HumaOptions{
		DocsPath:     "/reference",
		OpenAPIPath:  "/spec",
		SchemasPath:  "/schemas",
		DocsRenderer: httpx.DocsRendererScalar,
	})

	return httpx.New(
		httpx.WithAdapter(stdAdapter),
		httpx.WithBasePath("/api"),
		httpx.WithOpenAPIInfo("httpx organization example", "1.0.0", "Docs, security, and group defaults"),
		httpx.WithSecurity(httpx.SecurityOptions{
			Schemes: httpx.SecuritySchemes(map[string]*huma.SecurityScheme{
				"bearerAuth": {
					Type:   "http",
					Scheme: "bearer",
				},
			}),
			Requirements: httpx.SecurityRequirements(
				httpx.SecurityRequirement("bearerAuth"),
			),
		}),
	)
}

func registerOrganizationRoutes(server httpx.ServerRuntime) {
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

	admin := server.Group("/admin")
	configureAdminGroup(admin)
	httpx.MustGroupGet(admin, "/tenants/{id}", func(_ context.Context, input *tenantInput) (*tenantOutput, error) {
		out := &tenantOutput{}
		out.Body.ID = input.ID
		out.Body.Name = "tenant-" + input.ID
		return out, nil
	}, func(op *huma.Operation) {
		op.Summary = "Get tenant"
	})
}

func configureAdminGroup(admin *httpx.Group) {
	admin.RegisterTags(httpx.TagDefinitions(
		&huma.Tag{Name: "admin", Description: "Administrative APIs"},
		&huma.Tag{Name: "tenants", Description: "Tenant management"},
	))
	admin.DefaultTags(httpx.Tags("admin", "tenants"))
	admin.DefaultSecurity(httpx.SecurityRequirements(httpx.SecurityRequirement("bearerAuth")))
	admin.DefaultParameters(httpx.Parameters(&huma.Param{
		Name:        "X-Tenant",
		In:          "header",
		Description: "tenant scope",
		Schema:      &huma.Schema{Type: "string"},
	}))
	admin.DefaultSummaryPrefix("Admin")
	admin.DefaultDescription("Administrative operations with shared docs metadata")
	admin.DefaultExternalDocs(&huma.ExternalDocs{
		Description: "Admin handbook",
		URL:         "https://example.com/admin-handbook",
	})
	admin.DefaultExtensions(httpx.Extensions(map[string]any{
		"x-owner": "platform",
	}))
}
